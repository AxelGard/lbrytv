package api

import (
	"github.com/lbryio/lbrytv/app/proxy"
	"github.com/lbryio/lbrytv/app/publish"
	"github.com/lbryio/lbrytv/app/users"

	"github.com/gorilla/mux"
)

// InstallRoutes sets up global API handlers
func InstallRoutes(proxyService *proxy.Service, r *mux.Router) {
	authenticator := users.NewAuthenticator(users.NewUserService())
	proxyHandler := proxy.NewRequestHandler(proxyService)
	upHandler := publish.NewUploadHandler(publish.UploadOpts{ProxyService: proxyService})

	r.HandleFunc("/", Index)
	v1Router := r.PathPrefix("/api/v1").Subrouter()
	v1Router.HandleFunc("/proxy", authenticator.Wrap(upHandler.Handle)).MatcherFunc(upHandler.CanHandle)
	v1Router.HandleFunc("/proxy", captureErrors(proxyHandler.Handle))

	// TODO: For temporary backwards compatibility, remove after JS code has been updated to use paths above
	r.HandleFunc("/api/proxy", captureErrors(proxyHandler.Handle))
	r.HandleFunc("/api/proxy/", captureErrors(proxyHandler.Handle))

	r.HandleFunc("/content/claims/{uri}/{claim}/{filename}", captureErrors(ContentByClaimsURI)).Methods("GET")
	r.HandleFunc("/content/url", captureErrors(ContentByURL)).Methods("GET")
}
