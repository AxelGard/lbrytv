package proxy

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/lbryio/lbrytv/app/auth"
	"github.com/lbryio/lbrytv/app/sdkrouter"
	"github.com/lbryio/lbrytv/app/wallet"
	"github.com/lbryio/lbrytv/internal/monitor"
	"github.com/lbryio/lbrytv/internal/responses"
	"github.com/ybbus/jsonrpc"
)

// Handle forwards client JSON-RPC request to proxy.
func Handle(w http.ResponseWriter, r *http.Request) {
	var proxyHandlerLogger = monitor.NewModuleLogger("proxy_handlers")
	if r.Body == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("empty request body"))
		proxyHandlerLogger.Log().Errorf("empty request body")
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error reading request body"))
		proxyHandlerLogger.Log().Errorf("error reading request body: %v", err.Error())
		return
	}

	// We're in RPC-response-land from here on down
	responses.AddJSONContentType(w)

	var req jsonrpc.RPCRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		w.Write(NewJSONParseError(err).JSON())
		return
	}

	var userID int
	var sdkAddress string
	if MethodNeedsAuth(req.Method) {
		authResult := auth.FromRequest(r)
		if !authResult.AuthAttempted() {
			w.Write(NewAuthRequiredError(errors.New(responses.AuthRequiredErrorMessage)).JSON())
			return
		}
		if !authResult.Authenticated() {
			w.Write(NewForbiddenError(authResult.Err()).JSON())
			return
		}
		userID = authResult.User().ID
		sdkAddress = authResult.SDKAddress
	}

	rt := sdkrouter.FromRequest(r)

	if sdkAddress == "" {
		sdkAddress = rt.RandomServer().Address
	}

	c := NewCaller(sdkAddress, userID)
	w.Write(c.Call(&req))
}

// HandleCORS returns necessary CORS headers for pre-flight requests to proxy API
func HandleCORS(w http.ResponseWriter, r *http.Request) {
	hs := w.Header()
	hs.Set("Access-Control-Max-Age", "7200")
	hs.Set("Access-Control-Allow-Origin", "*")
	hs.Set("Access-Control-Allow-Headers", wallet.TokenHeader+", Origin, X-Requested-With, Content-Type, Accept")
	w.WriteHeader(http.StatusOK)
}
