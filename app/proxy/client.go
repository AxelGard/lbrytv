package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lbryio/lbrytv/internal/lbrynet"
	"github.com/lbryio/lbrytv/internal/metrics"
	"github.com/lbryio/lbrytv/internal/monitor"

	ljsonrpc "github.com/lbryio/lbry.go/v2/extras/jsonrpc"
	"github.com/ybbus/jsonrpc"
)

const walletLoadRetries = 3
const walletLoadRetryWait = time.Millisecond * 100

var ClientLogger = monitor.NewModuleLogger("proxy_client")

type LbrynetClient interface {
	Call(q *Query) (*jsonrpc.RPCResponse, error)
}

type Client struct {
	rpcClient jsonrpc.RPCClient
	endpoint  string
	wallet    string
	retries   int
}

func NewClient(endpoint string, wallet string, timeout time.Duration) LbrynetClient {
	return Client{
		endpoint: endpoint,
		rpcClient: jsonrpc.NewClientWithOpts(endpoint, &jsonrpc.RPCClientOpts{
			HTTPClient: &http.Client{Timeout: time.Second * timeout}}),
		wallet: wallet,
	}
}

func (c Client) Call(q *Query) (*jsonrpc.RPCResponse, error) {
	var (
		i        int
		r        *jsonrpc.RPCResponse
		err      error
		duration float64
	)

	callMetrics := metrics.ProxyCallDurations.WithLabelValues(q.Method(), c.endpoint)
	failureMetrics := metrics.ProxyCallFailedDurations.WithLabelValues(q.Method(), c.endpoint)

	for i = 0; i < walletLoadRetries; i++ {
		start := time.Now()

		r, err = c.rpcClient.CallRaw(q.Request)

		duration = time.Since(start).Seconds()
		callMetrics.Observe(duration)

		// Generally a HTTP transport failure (connect error etc)
		if err != nil {
			failureMetrics.Observe(duration)
			Logger.LogFailedQuery(q.Method(), duration, q.Params(), r.Error)
			return nil, err
		}

		// This checks if LbrynetServer responded with missing wallet error and tries to reload it,
		// then repeats the request again.
		if c.isWalletNotLoaded(r) {
			time.Sleep(walletLoadRetryWait)
			// We need to use Lbry JSON-RPC client here for easier request/response processing
			client := ljsonrpc.NewClient(c.endpoint)
			_, err := client.WalletAdd(c.wallet)
			if err != nil {
				ClientLogger.WithFields(monitor.F{
					"wallet_id": c.wallet, "endpoint": c.endpoint,
				}).Errorf("encountered an error adding wallet manually: %v", err)
			}

			// Alert sentry on the last failed wallet load attempt
			if i >= walletLoadRetries-1 {
				monitor.CaptureException(
					fmt.Errorf("gave up on manually adding wallet: %v", r.Error.Message), map[string]string{
						"wallet_id": c.wallet,
						"endpoint":  c.endpoint,
						"retries":   fmt.Sprintf("%v", i),
					})
			}
		} else if c.isWalletAlreadyLoaded(r) {
			continue
		} else {
			break
		}
	}

	if (r != nil && r.Error != nil) || err != nil {
		Logger.LogFailedQuery(q.Method(), duration, q.Params(), r.Error)
		failureMetrics.Observe(duration)
	} else {
		Logger.LogSuccessfulQuery(q.Method(), duration, q.Params(), r)
	}

	return r, err
}

func (c *Client) isWalletNotLoaded(r *jsonrpc.RPCResponse) bool {
	if r.Error != nil {
		wErr := lbrynet.NewWalletError(0, errors.New(r.Error.Message))
		if errors.As(wErr, &lbrynet.WalletNotLoaded{}) {
			return true
		}
	}
	return false
}

func (c *Client) isWalletAlreadyLoaded(r *jsonrpc.RPCResponse) bool {
	if r.Error != nil {
		wErr := lbrynet.NewWalletError(0, errors.New(r.Error.Message))
		if errors.As(wErr, &lbrynet.WalletAlreadyLoaded{}) {
			return true
		}
	}
	return false
}
