package query

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/lbryio/lbrytv/app/query/cache"
	"github.com/lbryio/lbrytv/app/rpcerrors"
	"github.com/lbryio/lbrytv/app/sdkrouter"
	"github.com/lbryio/lbrytv/app/wallet"
	"github.com/lbryio/lbrytv/config"
	"github.com/lbryio/lbrytv/internal/errors"
	"github.com/lbryio/lbrytv/internal/lbrynet"
	"github.com/lbryio/lbrytv/internal/metrics"
	"github.com/lbryio/lbrytv/internal/monitor"

	"github.com/sirupsen/logrus"
	"github.com/ybbus/jsonrpc"
)

const (
	walletLoadRetries   = 3
	walletLoadRetryWait = 100 * time.Millisecond
)

type Hook func(c *Caller, q *Query) (*jsonrpc.RPCResponse, error)
type hookEntry struct {
	method   string
	function Hook
}

// Caller patches through JSON-RPC requests from clients, doing pre/post-processing,
// account processing and validation.
type Caller struct {
	// Preprocessor is applied to query before it's sent to the SDK.
	Preprocessor   func(q *Query)
	preflightHooks []hookEntry

	// Cache stores cachable queries to improve performance
	Cache cache.QueryCache

	client   jsonrpc.RPCClient
	userID   int
	endpoint string
}

func NewCaller(endpoint string, userID int) *Caller {
	c := &Caller{
		client: jsonrpc.NewClientWithOpts(endpoint, &jsonrpc.RPCClientOpts{
			HTTPClient: &http.Client{
				Timeout: sdkrouter.RPCTimeout,
				Transport: &http.Transport{
					Dial: (&net.Dialer{
						Timeout:   120 * time.Second,
						KeepAlive: 120 * time.Second,
					}).Dial,
					TLSHandshakeTimeout:   30 * time.Second,
					ResponseHeaderTimeout: 300 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
				},
			},
		}),
		endpoint: endpoint,
		userID:   userID,
	}
	c.addDefaultHooks()
	return c
}

// AddPreflightHook adds query pre-flight hook function,
// allowing to amend the query before it gets sent to the JSON-RPC server,
// with an option to return an early response, avoiding sending the query
// to JSON-RPC server altogether
func (c *Caller) AddPreflightHook(method string, hf Hook) {
	c.preflightHooks = append(c.preflightHooks, hookEntry{method, hf})
	logger.Log().Debugf("added a preflight hook for method %v", method)
}

func (c *Caller) addDefaultHooks() {
	c.AddPreflightHook("", fromCache)
	c.AddPreflightHook("status", getStatusResponse)
	c.AddPreflightHook("get", preflightHookGet)
}

// Call method forwards a JSON-RPC request to the lbrynet server.
// It returns a response that is ready to be sent back to the JSON-RPC client as is.
func (c *Caller) Call(req *jsonrpc.RPCRequest) (*jsonrpc.RPCResponse, error) {
	walletID := ""
	if c.userID != 0 {
		walletID = sdkrouter.WalletID(c.userID)
	}

	q, err := NewQuery(req, walletID)
	if err != nil {
		return nil, err
	}

	var res *jsonrpc.RPCResponse

	// Applying preflight hooks
	for _, hook := range c.preflightHooks {
		if hook.method == "" || hook.method == q.Method() {
			res, err = hook.function(c, q)
			if err != nil {
				return nil, rpcerrors.NewSDKError(err)
			}
			if res != nil {
				return res, nil
			}
		}
	}

	if res == nil {
		res, err = c.callQueryWithRetry(q)
		if err != nil {
			return nil, rpcerrors.NewSDKError(err)
		}
	}

	if isCacheable(q) {
		c.Cache.Save(q.Method(), q.Params(), res)
	}

	return res, nil
}

func (c *Caller) callQueryWithRetry(q *Query) (*jsonrpc.RPCResponse, error) {
	var (
		r        *jsonrpc.RPCResponse
		err      error
		duration float64
	)

	for i := 0; i < walletLoadRetries; i++ {
		start := time.Now()

		r, err = c.client.CallRaw(q.Request)

		duration = time.Since(start).Seconds()
		metrics.ProxyCallDurations.WithLabelValues(q.Method(), c.endpoint).Observe(duration)

		// Generally a HTTP transport failure (connect error etc)
		if err != nil {
			logger.Log().Errorf("error sending query to %v: %v", c.endpoint, err)
			metrics.ProxyCallFailedDurations.WithLabelValues(q.Method(), c.endpoint, metrics.FailureKindNet).Observe(duration)
			return nil, errors.Err(err)
		}

		// This checks if LbrynetServer responded with missing wallet error and tries to reload it,
		// then repeats the request again.
		if isErrWalletNotLoaded(r) {
			time.Sleep(walletLoadRetryWait)
			// Using LBRY JSON-RPC client here for easier request/response processing
			err := wallet.LoadWallet(c.endpoint, c.userID)
			// Alert sentry on the last failed wallet load attempt
			if err != nil && i >= walletLoadRetries-1 {
				e := errors.Prefix("gave up manually adding wallet", err)
				logger.WithFields(logrus.Fields{
					"user_id":  c.userID,
					"endpoint": c.endpoint,
				}).Error(e)
				monitor.ErrorToSentry(e, map[string]string{
					"user_id":  fmt.Sprintf("%d", c.userID),
					"endpoint": c.endpoint,
					"retries":  fmt.Sprintf("%d", i),
				})
			}
		} else if isErrWalletAlreadyLoaded(r) {
			continue
		} else {
			break
		}
	}

	logFields := logrus.Fields{
		"method":   q.Method(),
		"params":   q.Params(),
		"endpoint": c.endpoint,
		"user_id":  c.userID,
		"duration": duration,
	}
	if err != nil || (r != nil && r.Error != nil) {
		logFields["response"] = r.Error
		logger.WithFields(logFields).Error("rpc call error")
		metrics.ProxyCallFailedDurations.WithLabelValues(q.Method(), c.endpoint, metrics.FailureKindRPC).Observe(duration)
	} else {
		if config.ShouldLogResponses() {
			logFields["response"] = r
		}
		logger.WithFields(logFields).Debug("rpc call processed")
	}

	return r, err
}

// isCacheable returns true if this query can be cached
func isCacheable(q *Query) bool {
	if q.Method() == MethodResolve && q.Params() != nil {
		paramsMap := q.Params().(map[string]interface{})
		if urls, ok := paramsMap[paramUrls].([]interface{}); ok {
			if len(urls) > cacheResolveLongerThan {
				return true
			}
		}
	} else if q.Method() == MethodClaimSearch {
		return true
	}
	return false
}

// fromCache returns cached response or nil in case it's a miss
func fromCache(c *Caller, q *Query) (*jsonrpc.RPCResponse, error) {
	if c.Cache == nil || !isCacheable(q) {
		return nil, nil
	}

	cached := c.Cache.Retrieve(q.Method(), q.Params())
	if cached == nil {
		return nil, nil
	}

	s, err := json.Marshal(cached)
	if err != nil {
		logger.Log().Errorf("error marshalling cached response")
		return nil, nil
	}

	response := q.newResponse()
	err = json.Unmarshal(s, &response)
	if err != nil {
		return nil, nil
	}

	logger.WithFields(logrus.Fields{"method": q.Method()}).Debug("cached query")
	return response, nil
}

func isErrWalletNotLoaded(r *jsonrpc.RPCResponse) bool {
	return r.Error != nil && errors.Is(lbrynet.NewWalletError(0, errors.Err(r.Error.Message)), lbrynet.ErrWalletNotLoaded)
}

func isErrWalletAlreadyLoaded(r *jsonrpc.RPCResponse) bool {
	return r.Error != nil && errors.Is(lbrynet.NewWalletError(0, errors.Err(r.Error.Message)), lbrynet.ErrWalletAlreadyLoaded)
}
