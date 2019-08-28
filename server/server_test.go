package server

import (
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/lbryio/lbrytv/app/proxy"

	"github.com/stretchr/testify/assert"
)

func TestStartAndServeUntilShutdown(t *testing.T) {
	server := NewServer(ServerOpts{
		Address:      "localhost:40080",
		ProxyService: proxy.NewService(""),
	})
	server.Start()
	go server.ServeUntilShutdown()

	response, err := http.Get("http://localhost:40080/")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, response.StatusCode)
	server.InterruptChan <- syscall.SIGINT

	// Retry 10 times to give the server a chance to shut down
	for range [10]int{} {
		time.Sleep(100 * time.Millisecond)
		response, err = http.Get("http://localhost:40080/")
		if err != nil {
			break
		}
	}
	assert.Error(t, err)
}

func TestHeaders(t *testing.T) {
	var (
		err      error
		response *http.Response
	)

	server := NewServer(ServerOpts{
		Address:      "localhost:40080",
		ProxyService: proxy.NewService(""),
	})
	server.Start()
	go server.ServeUntilShutdown()

	request, _ := http.NewRequest("OPTIONS", "http://localhost:40080/api/proxy", nil)
	client := http.Client{}

	// Retry 10 times to give the server a chance to start
	for range [10]int{} {
		time.Sleep(100 * time.Millisecond)
		response, err = client.Do(request)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "*", response.Header["Access-Control-Allow-Origin"][0])
	assert.Equal(t, "X-Lbry-Auth-Token, Origin, X-Requested-With, Content-Type, Accept", response.Header["Access-Control-Allow-Headers"][0])
	server.InterruptChan <- syscall.SIGINT
}
