package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/lbryio/lbrytv/apps/collector/models"
	env "github.com/lbryio/lbrytv/apps/environment"
	"github.com/lbryio/lbrytv/internal/responses"
	"github.com/markbates/pkger"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gorilla/mux"
)

func RouteInstaller(r *mux.Router, _ *env.Environment) {
	v1Router := r.PathPrefix("/api/v1").Subrouter()

	v1Router.HandleFunc("/events/video", EventHandler).Methods(http.MethodPost)
}

func EventHandler(w http.ResponseWriter, r *http.Request) {
	responses.AddJSONContentType(w)
	// resp := make(map[string]string)
	code := http.StatusOK

	f, err := pkger.Open("/apps/collector/openapi.yml")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error opening file for pkger: %v", err)
		return
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	swagger, _ := openapi3.NewSwaggerLoader().LoadSwaggerFromData(data)
	router := openapi3filter.NewRouter().WithSwagger(swagger)
	route, pathParams, err := router.FindRoute(r.Method, r.URL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "route error: %v", err)
		return
	}
	validator := &openapi3filter.RequestValidationInput{
		Request:    r,
		PathParams: pathParams,
		Route:      route,
	}
	if err := openapi3filter.ValidateRequest(context.Background(), validator); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "validation error: %v", err)
		return
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error reading request: %v", err)
		return
	}

	event := models.Event{}
	err = json.Unmarshal(reqBody, &event)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error marshaling event: %v", err)
	}

	err = event.InsertG(boil.Infer())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error saving event: %v", err)
	}

	w.WriteHeader(code)
}
