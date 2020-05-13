package sdkrouter

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

type ctxKey int

const contextKey ctxKey = iota

func FromRequest(r *http.Request) *Router {
	v := r.Context().Value(contextKey)
	if v == nil {
		panic("sdkrouter.Middleware is required")
	}
	return v.(*Router)
}

func AddToRequest(rt *Router, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r.Clone(context.WithValue(r.Context(), contextKey, rt)))
	}
}

func Middleware(rt *Router) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return AddToRequest(rt, next.ServeHTTP)
	}
}
