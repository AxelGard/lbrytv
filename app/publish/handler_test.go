package publish

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/lbryio/lbrytv/app/users"
	"github.com/lbryio/lbrytv/internal/lbrynet"
	"github.com/ybbus/jsonrpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DummyPublisher struct {
	called    bool
	filePath  string
	accountID string
	rawQuery  []byte
}

func (p *DummyPublisher) Publish(filePath, accountID string, rawQuery []byte) []byte {
	p.called = true
	p.filePath = filePath
	p.accountID = accountID
	p.rawQuery = rawQuery
	return []byte(lbrynet.ExampleStreamCreateResponse)
}

func TestUploadHandler(t *testing.T) {
	req := CreatePublishRequest(t, []byte("test file"))
	req.Header.Set(users.TokenHeader, "uPldrToken")

	rr := httptest.NewRecorder()
	authenticator := users.NewAuthenticator(&users.TestUserRetriever{WalletID: "UPldrAcc", Token: "uPldrToken"})
	publisher := &DummyPublisher{}
	pubHandler, err := NewUploadHandler(UploadOpts{Path: os.TempDir(), Publisher: publisher})
	assert.Nil(t, err)

	http.HandlerFunc(authenticator.Wrap(pubHandler.Handle)).ServeHTTP(rr, req)
	response := rr.Result()
	respBody, _ := ioutil.ReadAll(response.Body)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, lbrynet.ExampleStreamCreateResponse, string(respBody))

	require.True(t, publisher.called)
	expectedPath := path.Join(os.TempDir(), "UPldrAcc", ".*_lbry_auto_test_file")
	assert.Regexp(t, expectedPath, publisher.filePath)
	assert.Equal(t, "UPldrAcc", publisher.accountID)
	assert.Equal(t, lbrynet.ExampleStreamCreateRequest, string(publisher.rawQuery))

	_, err = os.Stat(publisher.filePath)
	assert.True(t, os.IsNotExist(err))
}

func TestUploadHandlerAuthRequired(t *testing.T) {
	var rpcResponse jsonrpc.RPCResponse
	req := CreatePublishRequest(t, []byte("test file"))

	rr := httptest.NewRecorder()
	authenticator := users.NewAuthenticator(&users.TestUserRetriever{})
	publisher := &DummyPublisher{}
	pubHandler, err := NewUploadHandler(UploadOpts{Path: os.TempDir(), Publisher: publisher})
	assert.Nil(t, err)

	http.HandlerFunc(authenticator.Wrap(pubHandler.Handle)).ServeHTTP(rr, req)
	response := rr.Result()

	assert.Equal(t, http.StatusOK, response.StatusCode)
	err = json.Unmarshal(rr.Body.Bytes(), &rpcResponse)
	require.Nil(t, err)
	assert.Equal(t, "authentication required", rpcResponse.Error.Message)
	require.False(t, publisher.called)
}

func TestUploadHandlerSystemError(t *testing.T) {
	var rpcResponse jsonrpc.RPCResponse

	// Creating POST data manually here because we need to avoid writer.Close()
	data := []byte("test file")
	readSeeker := bytes.NewReader(data)
	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)

	fileBody, err := writer.CreateFormFile(FileFieldName, "lbry_auto_test_file")
	require.Nil(t, err)
	io.Copy(fileBody, readSeeker)

	jsonPayload, err := writer.CreateFormField(JSONRPCFieldName)
	require.Nil(t, err)
	jsonPayload.Write([]byte(lbrynet.ExampleStreamCreateRequest))

	// <--- Not calling writer.Close() here to create an unexpected EOF

	req, err := http.NewRequest("POST", "/", bytes.NewReader(body.Bytes()))
	require.Nil(t, err)

	req.Header.Set(users.TokenHeader, "uPldrToken")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	authenticator := users.NewAuthenticator(&users.TestUserRetriever{WalletID: "UPldrAcc", Token: "uPldrToken"})
	publisher := &DummyPublisher{}
	pubHandler, err := NewUploadHandler(UploadOpts{Path: os.TempDir(), Publisher: publisher})
	assert.Nil(t, err)

	http.HandlerFunc(authenticator.Wrap(pubHandler.Handle)).ServeHTTP(rr, req)
	response := rr.Result()

	require.False(t, publisher.called)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	err = json.Unmarshal(rr.Body.Bytes(), &rpcResponse)
	require.Nil(t, err)
	assert.Equal(t, "unexpected EOF", rpcResponse.Error.Message)
	require.False(t, publisher.called)
}

func TestNewUploadHandler(t *testing.T) {
	h, err := NewUploadHandler(UploadOpts{})
	assert.Error(t, err, "need either a ProxyService or a Publisher instance")
	assert.Nil(t, h)
}
