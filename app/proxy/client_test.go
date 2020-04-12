package proxy

import (
	"math/rand"
	"testing"
	"time"

	"github.com/lbryio/lbrytv/app/sdkrouter"
	"github.com/lbryio/lbrytv/config"
	"github.com/lbryio/lbrytv/internal/lbrynet"

	"github.com/stretchr/testify/require"
	"github.com/ybbus/jsonrpc"
)

type MockRPCClient struct {
	Delay        time.Duration
	LastRequest  jsonrpc.RPCRequest
	NextResponse chan *jsonrpc.RPCResponse
}

func NewMockRPCClient() *MockRPCClient {
	return &MockRPCClient{
		NextResponse: make(chan *jsonrpc.RPCResponse, 100),
	}
}

func (c MockRPCClient) AddNextResponse(r *jsonrpc.RPCResponse) {
	c.NextResponse <- r
}

func (c MockRPCClient) Call(method string, params ...interface{}) (*jsonrpc.RPCResponse, error) {
	return <-c.NextResponse, nil
}

func (c *MockRPCClient) CallRaw(request *jsonrpc.RPCRequest) (*jsonrpc.RPCResponse, error) {
	c.LastRequest = *request
	return <-c.NextResponse, nil
}

func (c MockRPCClient) CallFor(out interface{}, method string, params ...interface{}) error {
	return nil
}

func (c MockRPCClient) CallBatch(requests jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error) {
	return nil, nil
}

func (c MockRPCClient) CallBatchRaw(requests jsonrpc.RPCRequests) (jsonrpc.RPCResponses, error) {
	return nil, nil
}

func TestClientCallDoesReloadWallet(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	dummyUserID := rand.Intn(100)
	rt := sdkrouter.New(config.GetLbrynetServers())

	wid, _ := lbrynet.InitializeWallet(rt, dummyUserID)
	err := lbrynet.UnloadWallet(rt, dummyUserID)
	require.NoError(t, err)

	c := NewClient(rt.GetServer(wid).Address, wid, time.Second*1)

	q, _ := NewQuery(newRawRequest(t, "wallet_balance", nil))
	q.SetWalletID(wid)
	r, err := c.Call(q)

	// err = json.Unmarshal(result, response)
	require.NoError(t, err)
	require.Nil(t, r.Error)
}

func TestClientCallDoesNotReloadWalletAfterOtherErrors(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	walletID := sdkrouter.WalletID(rand.Intn(100))

	mc := NewMockRPCClient()
	c := &Client{rpcClient: mc}
	q, _ := NewQuery(newRawRequest(t, "wallet_balance", nil))
	q.SetWalletID(walletID)

	mc.AddNextResponse(&jsonrpc.RPCResponse{
		JSONRPC: "2.0",
		Error: &jsonrpc.RPCError{
			Message: "Couldn't find wallet: //",
		},
	})
	mc.AddNextResponse(&jsonrpc.RPCResponse{
		JSONRPC: "2.0",
		Error: &jsonrpc.RPCError{
			Message: "Wallet at path // was not found",
		},
	})

	r, err := c.Call(q)
	require.NoError(t, err)
	require.Equal(t, "Wallet at path // was not found", r.Error.Message)
}

func TestClientCallDoesNotReloadWalletIfAlreadyLoaded(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	wid := sdkrouter.WalletID(rand.Intn(100))

	mc := NewMockRPCClient()
	c := &Client{rpcClient: mc}
	q, _ := NewQuery(newRawRequest(t, "wallet_balance", nil))
	q.SetWalletID(wid)

	mc.AddNextResponse(&jsonrpc.RPCResponse{
		JSONRPC: "2.0",
		Error: &jsonrpc.RPCError{
			Message: "Couldn't find wallet: //",
		},
	})
	mc.AddNextResponse(&jsonrpc.RPCResponse{
		JSONRPC: "2.0",
		Error: &jsonrpc.RPCError{
			Message: "Wallet at path // is already loaded",
		},
	})
	mc.AddNextResponse(&jsonrpc.RPCResponse{
		JSONRPC: "2.0",
		Result:  `"99999.00"`,
	})

	r, err := c.Call(q)

	require.NoError(t, err)
	require.Nil(t, r.Error)
	require.Equal(t, `"99999.00"`, r.Result)
}
