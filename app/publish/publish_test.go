package publish

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/lbryio/lbrytv/app/sdkrouter"
	"github.com/lbryio/lbrytv/app/wallet"
	"github.com/lbryio/lbrytv/config"
	"github.com/lbryio/lbrytv/internal/responses"
	"github.com/lbryio/lbrytv/internal/storage"
	"github.com/lbryio/lbrytv/internal/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func copyToDocker(t *testing.T, fileName string) {
	cmd := fmt.Sprintf(`docker cp %v lbrytv_lbrynet_1:/storage`, fileName)
	if _, err := exec.Command("bash", "-c", cmd).Output(); err != nil {
		t.Skipf("skipping TestLbrynetPublisher (cannot copy %v to docker container: %v)", fileName, err)
	}
}

func TestLbrynetPublisher(t *testing.T) {
	// dummyUserID := 751365
	authToken := "zzz"

	dbConfig := config.GetDatabase()
	params := storage.ConnParams{
		Connection: dbConfig.Connection,
		DBName:     dbConfig.DBName,
		Options:    dbConfig.Options,
	}
	c, connCleanup := storage.CreateTestConn(params)
	c.SetDefaultConnection()
	defer connCleanup()

	reqChan := test.ReqChan()
	ts := test.MockHTTPServer(reqChan)
	defer ts.Close()
	go func() {
		req := <-reqChan
		responses.AddJSONContentType(req.W)
		ts.NextResponse <- fmt.Sprintf(`{
			"success": true,
			"error": null,
			"data": {
			  "user_id": %v,
			  "has_verified_email": true
			}
		}`, 751365)
	}()

	rt := sdkrouter.New(config.GetLbrynetServers())
	p := &LbrynetPublisher{rt}
	u, err := wallet.GetUserWithWallet(rt, ts.URL, authToken, "")
	require.NoError(t, err)

	data := []byte("test file")
	f, err := ioutil.TempFile(os.TempDir(), "*")
	require.NoError(t, err)
	_, err = f.Write(data)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
	defer os.Remove(f.Name())

	copyToDocker(t, f.Name())

	query := []byte(`{
		"jsonrpc": "2.0",
		"method": "stream_create",
		"params": {
			"name": "test",
			"title": "test",
			"description": "test description",
			"bid": "0.000001",
			"languages": [
				"en"
			],
			"tags": [],
			"thumbnail_url": "http://smallmedia.com/thumbnail.jpg",
			"license": "None",
			"release_time": 1567580184,
			"file_path": "__POST_FILE__"
		},
		"id": 1567580184168
	}`)

	rawResp := p.Publish(path.Join("/storage", path.Base(f.Name())), u.ID, query)

	// This is all we can check for now without running on testnet or crediting some funds to the test account
	assert.Regexp(t, "Not enough funds to cover this transaction", string(rawResp))
}
