package query

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/lbryio/lbrytv/app/sdkrouter"
	"github.com/lbryio/lbrytv/app/wallet"
	"github.com/lbryio/lbrytv/internal/test"

	ljsonrpc "github.com/lbryio/lbry.go/v2/extras/jsonrpc"

	"github.com/davecgh/go-spew/spew"
	logrusTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ybbus/jsonrpc"
)

func TestCaller_CallRelaxedMethods(t *testing.T) {
	reqChan := test.ReqChan()
	srv := test.MockHTTPServer(reqChan)
	defer srv.Close()
	caller := NewCaller(srv.URL, 0)

	for _, m := range relaxedMethods {
		if m == MethodStatus {
			continue
		}
		t.Run(m, func(t *testing.T) {
			srv.NextResponse <- test.EmptyResponse()
			resp := caller.Call(jsonrpc.NewRequest(m))
			require.NotEqual(t, "authentication required", test.StrToRes(t, string(resp)).Error.Message)

			receivedRequest := <-reqChan
			expectedRequest := test.ReqToStr(t, jsonrpc.RPCRequest{
				Method:  m,
				Params:  nil,
				JSONRPC: "2.0",
			})
			assert.EqualValues(t, expectedRequest, receivedRequest.Body)
		})
	}
}

func TestCaller_CallAmbivalentMethodsWithoutWallet(t *testing.T) {
	reqChan := test.ReqChan()
	srv := test.MockHTTPServer(reqChan)
	defer srv.Close()
	caller := NewCaller(srv.URL, 0)

	for _, m := range relaxedMethods {
		if !methodInList(m, walletSpecificMethods) {
			continue
		}
		t.Run(m, func(t *testing.T) {
			srv.NextResponse <- test.EmptyResponse()
			resp := caller.Call(jsonrpc.NewRequest(m))
			require.NotEqual(t, "authentication required", test.StrToRes(t, string(resp)).Error.Message)

			receivedRequest := <-reqChan
			expectedRequest := test.ReqToStr(t, jsonrpc.RPCRequest{
				Method:  m,
				Params:  nil,
				JSONRPC: "2.0",
			})
			assert.EqualValues(t, expectedRequest, receivedRequest.Body)
		})
	}
}

func TestCaller_CallAmbivalentMethodsWithWallet(t *testing.T) {
	reqChan := test.ReqChan()
	srv := test.MockHTTPServer(reqChan)
	defer srv.Close()
	dummyUserID := 123321
	authedCaller := NewCaller(srv.URL, dummyUserID)
	var methodsTested int

	for _, m := range relaxedMethods {
		if !methodInList(m, walletSpecificMethods) {
			continue
		}
		methodsTested++
		t.Run(m, func(t *testing.T) {
			srv.NextResponse <- test.EmptyResponse()
			resp := authedCaller.Call(jsonrpc.NewRequest(m))
			require.NotEqual(t, "authentication required", test.StrToRes(t, string(resp)).Error.Message)

			receivedRequest := <-reqChan
			expectedRequest := test.ReqToStr(t, jsonrpc.RPCRequest{
				Method: m,
				Params: map[string]interface{}{
					"wallet_id": sdkrouter.WalletID(dummyUserID),
				},
				JSONRPC: "2.0",
			})
			assert.EqualValues(t, expectedRequest, receivedRequest.Body)
		})
	}

	if methodsTested == 0 {
		t.Fatal("no ambivalent methods found, that doesn't look right")
	}
}

func TestCaller_CallNonRelaxedMethods(t *testing.T) {
	reqChan := test.ReqChan()
	srv := test.MockHTTPServer(reqChan)
	defer srv.Close()

	caller := NewCaller(srv.URL, 0)
	for _, m := range walletSpecificMethods {
		if methodInList(m, relaxedMethods) {
			continue
		}
		t.Run(m, func(t *testing.T) {
			resp := caller.Call(jsonrpc.NewRequest(m))
			select {
			case <-reqChan:
				t.Fatal("a request went throught")
			default:
			}
			assert.Equal(t, "authentication required", test.StrToRes(t, string(resp)).Error.Message)
		})
	}
}

func TestCaller_CallForbiddenMethod(t *testing.T) {
	caller := NewCaller(test.RandServerAddress(t), 0)
	result := caller.Call(jsonrpc.NewRequest("stop"))
	assert.Contains(t, string(result), `"message": "forbidden method"`)
}

func TestCaller_CallAttachesWalletID(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	dummyUserID := 123321

	reqChan := test.ReqChan()
	srv := test.MockHTTPServer(reqChan)
	defer srv.Close()
	srv.NextResponse <- test.EmptyResponse()
	caller := NewCaller(srv.URL, dummyUserID)
	caller.Call(jsonrpc.NewRequest("channel_create", map[string]interface{}{"name": "test", "bid": "0.1"}))
	receivedRequest := <-reqChan

	expectedRequest := test.ReqToStr(t, jsonrpc.RPCRequest{
		Method: "channel_create",
		Params: map[string]interface{}{
			"name":      "test",
			"bid":       "0.1",
			"wallet_id": sdkrouter.WalletID(dummyUserID),
		},
		JSONRPC: "2.0",
	})
	assert.EqualValues(t, expectedRequest, receivedRequest.Body)
}

func TestCaller_SetPreprocessor(t *testing.T) {
	reqChan := test.ReqChan()
	srv := test.MockHTTPServer(reqChan)
	defer srv.Close()

	c := NewCaller(srv.URL, 0)

	c.Preprocessor = func(q *Query) {
		params := q.ParamsAsMap()
		if params == nil {
			q.Request.Params = map[string]string{"param": "123"}
		} else {
			params["param"] = "123"
			q.Request.Params = params
		}
	}

	srv.NextResponse <- test.EmptyResponse()

	c.Call(jsonrpc.NewRequest(relaxedMethods[0]))
	req := <-reqChan
	lastRequest := test.StrToReq(t, req.Body)

	p, ok := lastRequest.Params.(map[string]interface{})
	assert.True(t, ok, req.Body)
	assert.Equal(t, "123", p["param"], req.Body)
}

func TestCaller_CallSDKError(t *testing.T) {
	srv := test.MockHTTPServer(nil)
	defer srv.Close()
	srv.NextResponse <- `
		{
			"jsonrpc": "2.0",
			"error": {
			  "code": -32500,
			  "message": "After successfully executing the command, failed to encode result for JSON RPC response.",
			  "data": [
				"Traceback (most recent call last):",
				"  File \"lbry/extras/daemon/Daemon.py\", line 482, in handle_old_jsonrpc",
				"  File \"lbry/extras/daemon/Daemon.py\", line 235, in jsonrpc_dumps_pretty",
				"  File \"json/__init__.py\", line 238, in dumps",
				"  File \"json/encoder.py\", line 201, in encode",
				"  File \"json/encoder.py\", line 431, in _iterencode",
				"  File \"json/encoder.py\", line 405, in _iterencode_dict",
				"  File \"json/encoder.py\", line 405, in _iterencode_dict",
				"  File \"json/encoder.py\", line 325, in _iterencode_list",
				"  File \"json/encoder.py\", line 438, in _iterencode",
				"  File \"lbry/extras/daemon/json_response_encoder.py\", line 118, in default",
				"  File \"lbry/extras/daemon/json_response_encoder.py\", line 151, in encode_output",
				"  File \"torba/client/baseheader.py\", line 75, in __getitem__",
				"  File \"lbry/wallet/header.py\", line 35, in deserialize",
				"struct.error: unpack requires a buffer of 4 bytes",
				""
			  ]
			},
			"id": 0
		}`

	c := NewCaller(srv.URL, 0)
	hook := logrusTest.NewLocal(logger.Entry.Logger)
	response := c.Call(jsonrpc.NewRequest("resolve", map[string]interface{}{"urls": "what"}))
	var rpcResponse jsonrpc.RPCResponse
	json.Unmarshal(response, &rpcResponse)
	assert.Equal(t, rpcResponse.Error.Code, -32500)
	assert.Equal(t, "query", hook.LastEntry().Data["module"])
	assert.Equal(t, "resolve", hook.LastEntry().Data["method"])
}

func TestCaller_ClientJSONError(t *testing.T) {
	ts := test.MockHTTPServer(nil)
	defer ts.Close()
	ts.NextResponse <- `{"method":"version}` // note the missing close quote after "version

	c := NewCaller(ts.URL, 0)
	response := c.CallRaw([]byte(""))
	spew.Dump(response)
	var rpcResponse jsonrpc.RPCResponse
	json.Unmarshal(response, &rpcResponse)
	assert.Equal(t, "2.0", rpcResponse.JSONRPC)
	assert.Equal(t, -32700, rpcResponse.Error.Code)
	assert.Equal(t, "unexpected end of JSON input", rpcResponse.Error.Message)
}

func TestCaller_CallRaw(t *testing.T) {
	c := NewCaller(test.RandServerAddress(t), 0)
	for _, rawQ := range []string{``, ` `, `[]`, `[{}]`, `[""]`, `""`, `" "`} {
		t.Run(rawQ, func(t *testing.T) {
			r := c.CallRaw([]byte(rawQ))
			assert.Contains(t, string(r), `"code": -32700`, `raw query: `+rawQ)
		})
	}
	for _, rawQ := range []string{`{}`, `{"method": " "}`} {
		t.Run(rawQ, func(t *testing.T) {
			r := c.CallRaw([]byte(rawQ))
			assert.Contains(t, string(r), `"code": -32080`, `raw query: `+rawQ)
		})
	}
}

func TestCaller_Resolve(t *testing.T) {
	resolvedURL := "what#6769855a9aa43b67086f9ff3c1a5bacb5698a27a"
	resolvedClaimID := "6769855a9aa43b67086f9ff3c1a5bacb5698a27a"

	request := jsonrpc.NewRequest("resolve", map[string]interface{}{"urls": resolvedURL})
	rawCallResponse := NewCaller(test.RandServerAddress(t), 0).Call(request)

	var errorResponse jsonrpc.RPCResponse
	err := json.Unmarshal(rawCallResponse, &errorResponse)
	require.NoError(t, err)
	require.Nil(t, errorResponse.Error)

	var resolveResponse ljsonrpc.ResolveResponse
	parseRawResponse(t, rawCallResponse, &resolveResponse)
	assert.Equal(t, resolvedClaimID, resolveResponse[resolvedURL].ClaimID)
}

func TestCaller_WalletBalance(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	dummyUserID := rand.Intn(10^6-10^3) + 10 ^ 3

	request := jsonrpc.NewRequest("wallet_balance")

	result := NewCaller(test.RandServerAddress(t), 0).Call(request)
	assert.Contains(t, string(result), `"message": "authentication required"`)

	addr := test.RandServerAddress(t)
	err := wallet.Create(addr, dummyUserID)
	require.NoError(t, err)

	hook := logrusTest.NewLocal(logger.Entry.Logger)
	result = NewCaller(addr, dummyUserID).Call(request)

	var accountBalanceResponse struct {
		Available string `json:"available"`
	}
	parseRawResponse(t, result, &accountBalanceResponse)
	assert.EqualValues(t, "0.0", accountBalanceResponse.Available)
	assert.Equal(t, map[string]interface{}{"wallet_id": sdkrouter.WalletID(dummyUserID)}, hook.LastEntry().Data["params"])
	assert.Equal(t, "wallet_balance", hook.LastEntry().Data["method"])
}

func TestCaller_CallQueryWithRetry(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	dummyUserID := rand.Intn(100)
	addr := test.RandServerAddress(t)

	err := wallet.Create(addr, dummyUserID)
	require.NoError(t, err)
	err = wallet.UnloadWallet(addr, dummyUserID)
	require.NoError(t, err)

	q, err := NewQuery(jsonrpc.NewRequest("wallet_balance"), sdkrouter.WalletID(dummyUserID))
	require.NoError(t, err)

	// check that sdk loads the wallet and retries the query if the wallet was not initially loaded

	c := NewCaller(addr, dummyUserID)
	r, err := c.callQueryWithRetry(q)
	require.NoError(t, err)
	require.Nil(t, r.Error)
}

func TestCaller_DontReloadWalletAfterOtherErrors(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	walletID := sdkrouter.WalletID(rand.Intn(100))

	srv := test.MockHTTPServer(nil)
	defer srv.Close()

	c := NewCaller(srv.URL, 0)
	q, err := NewQuery(jsonrpc.NewRequest("wallet_balance"), walletID)
	require.NoError(t, err)

	srv.QueueResponses(
		test.ResToStr(t, jsonrpc.RPCResponse{
			JSONRPC: "2.0",
			Error: &jsonrpc.RPCError{
				Message: "Couldn't find wallet: //",
			},
		}),
		test.EmptyResponse(), // for the wallet_add call
		test.ResToStr(t, jsonrpc.RPCResponse{
			JSONRPC: "2.0",
			Error: &jsonrpc.RPCError{
				Message: "Wallet at path // was not found",
			},
		}),
	)

	r, err := c.callQueryWithRetry(q)
	require.NoError(t, err)
	require.Equal(t, "Wallet at path // was not found", r.Error.Message)
}

func TestCaller_DontReloadWalletIfAlreadyLoaded(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	walletID := sdkrouter.WalletID(rand.Intn(100))

	srv := test.MockHTTPServer(nil)
	defer srv.Close()

	c := NewCaller(srv.URL, 0)
	q, err := NewQuery(jsonrpc.NewRequest("wallet_balance"), walletID)
	require.NoError(t, err)

	srv.QueueResponses(
		test.ResToStr(t, jsonrpc.RPCResponse{
			JSONRPC: "2.0",
			Error: &jsonrpc.RPCError{
				Message: "Couldn't find wallet: //",
			},
		}),
		test.EmptyResponse(), // for the wallet_add call
		test.ResToStr(t, jsonrpc.RPCResponse{
			JSONRPC: "2.0",
			Error: &jsonrpc.RPCError{
				Message: "Wallet at path // is already loaded",
			},
		}),
		test.ResToStr(t, jsonrpc.RPCResponse{
			JSONRPC: "2.0",
			Result:  `"99999.00"`,
		}),
	)

	r, err := c.callQueryWithRetry(q)

	require.NoError(t, err)
	require.Nil(t, r.Error)
	require.Equal(t, `"99999.00"`, r.Result)
}

func TestSDKMethodStatus(t *testing.T) {
	c := NewCaller(test.RandServerAddress(t), 0)
	callResult := c.Call(jsonrpc.NewRequest("status"))
	var rpcResponse jsonrpc.RPCResponse
	json.Unmarshal(callResult, &rpcResponse)
	assert.Equal(t,
		"692EAWhtoqDuAfQ6KHMXxFxt8tkhmt7sfprEMHWKjy5hf6PwZcHDV542VHqRnFnTCD",
		rpcResponse.Result.(map[string]interface{})["installation_id"].(string))
}

func parseRawResponse(t *testing.T, rawCallResponse []byte, v interface{}) {
	assert.NotNil(t, rawCallResponse)
	var res jsonrpc.RPCResponse
	err := json.Unmarshal(rawCallResponse, &res)
	require.NoError(t, err)
	err = res.GetObject(v)
	require.NoError(t, err)
}
