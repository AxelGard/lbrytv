package proxy

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/lbryio/lbrytv/app/auth"
	"github.com/lbryio/lbrytv/app/sdkrouter"
	"github.com/lbryio/lbrytv/config"
	"github.com/lbryio/lbrytv/internal/storage"
	"github.com/lbryio/lbrytv/internal/test"
	"github.com/lbryio/lbrytv/models"

	ljsonrpc "github.com/lbryio/lbry.go/v2/extras/jsonrpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ybbus/jsonrpc"
)

const testSetupWait = 200 * time.Millisecond

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	dbConfig := config.GetDatabase()
	params := storage.ConnParams{
		Connection: dbConfig.Connection,
		DBName:     dbConfig.DBName,
		Options:    dbConfig.Options,
	}
	dbConn, connCleanup := storage.CreateTestConn(params)
	dbConn.SetDefaultConnection()

	code := m.Run()

	connCleanup()
	os.Exit(code)
}

func testFuncSetup() {
	storage.Conn.Truncate([]string{"users"})
	time.Sleep(testSetupWait)
}

func TestWithWrongAuthToken(t *testing.T) {
	testFuncSetup()

	ts := test.MockHTTPServer(nil)
	defer ts.Close()
	ts.NextResponse <- `{
		"success": false,
		"error": "could not authenticate user",
		"data": null
	}`

	q := jsonrpc.NewRequest("account_list")
	qBody, err := json.Marshal(q)
	require.NoError(t, err)
	r, err := http.NewRequest("POST", "/api/v1/proxy", bytes.NewBuffer(qBody))
	require.NoError(t, err)
	r.Header.Add("X-Lbry-Auth-Token", "xXxXxXx")

	rr := httptest.NewRecorder()
	rt := sdkrouter.New(config.GetLbrynetServers())
	handler := sdkrouter.Middleware(rt)(auth.Middleware(auth.NewIAPIProvider(rt, ts.URL))(http.HandlerFunc(Handle)))
	handler.ServeHTTP(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response jsonrpc.RPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "authentication error: could not authenticate user", response.Error.Message)
}

func TestAuthEmailNotVerified(t *testing.T) {
	testFuncSetup()

	ts := test.MockHTTPServer(nil)
	defer ts.Close()
	ts.NextResponse <- `{
	"success": true,
	"error": null,
	"data": {
		"user_id": 123,
		"has_verified_email": false
  	}
}`

	q := jsonrpc.NewRequest("account_list")
	qBody, err := json.Marshal(q)
	require.NoError(t, err)
	r, err := http.NewRequest("POST", "/api/v1/proxy", bytes.NewBuffer(qBody))
	require.NoError(t, err)
	r.Header.Add("X-Lbry-Auth-Token", "x")

	rr := httptest.NewRecorder()
	rt := sdkrouter.New(config.GetLbrynetServers())
	handler := sdkrouter.Middleware(rt)(auth.Middleware(auth.NewIAPIProvider(rt, ts.URL))(http.HandlerFunc(Handle)))
	handler.ServeHTTP(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response jsonrpc.RPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "must authenticate", response.Error.Message)
}

var nilProvider = func(token, ip string) (*models.User, error) { return nil, nil }

func TestWithoutToken(t *testing.T) {
	testFuncSetup()

	q, err := json.Marshal(jsonrpc.NewRequest("status"))
	require.NoError(t, err)
	r, err := http.NewRequest("POST", "/api/v1/proxy", bytes.NewBuffer(q))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	rt := sdkrouter.New(config.GetLbrynetServers())
	handler := sdkrouter.Middleware(rt)(auth.Middleware(nilProvider)(http.HandlerFunc(Handle)))
	handler.ServeHTTP(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)
	var response jsonrpc.RPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Nil(t, response.Error)

	var statusResponse ljsonrpc.StatusResponse
	err = ljsonrpc.Decode(response.Result, &statusResponse)
	require.NoError(t, err)
	assert.True(t, statusResponse.IsRunning)
}

func TestAccountSpecificWithoutToken(t *testing.T) {
	testFuncSetup()

	q := jsonrpc.NewRequest("account_list")
	qBody, err := json.Marshal(q)
	require.NoError(t, err)
	r, err := http.NewRequest("POST", "/api/v1/proxy", bytes.NewBuffer(qBody))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	rt := sdkrouter.New(config.GetLbrynetServers())
	handler := sdkrouter.Middleware(rt)(auth.Middleware(nilProvider)(http.HandlerFunc(Handle)))
	handler.ServeHTTP(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)
	var response jsonrpc.RPCResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.NotNil(t, response.Error)
	require.Equal(t, "authentication required", response.Error.Message)
}
