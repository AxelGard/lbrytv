package api

import (
	"github.com/lbryio/lbrytv/app/proxy"

	"github.com/gorilla/mux"
)

// InstallRoutes sets up global API handlers
func InstallRoutes(ps *proxy.Service, r *mux.Router) {
	r.HandleFunc("/", Index)

	proxyServer := proxy.NewRequestServer(ps)

	r.HandleFunc("/api/proxy", captureErrors(proxyServer.Handle))
	r.HandleFunc("/api/proxy/", captureErrors(proxyServer.Handle))
	r.HandleFunc("/content/claims/{uri}/{claim}/{filename}", captureErrors(ContentByClaimsURI))
	r.HandleFunc("/content/url", captureErrors(ContentByURL))
}
