package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/blevesearch/bleve"
	bleveHttp "github.com/blevesearch/bleve/http"
	"github.com/blevesearch/bleve/search/query"
	"github.com/gorilla/mux"
)

func httpServer() {

	// create a router to serve static files
	router := staticFileRouter()

	// add the API
	bleveHttp.RegisterIndexName(appName, index)
	//searchHandler := bleveHttp.NewSearchHandler(appName)
	searchHandler := &SearchHandler{}
	router.Handle("/api/search", searchHandler).Methods("POST")
	listFieldsHandler := bleveHttp.NewListFieldsHandler(appName)
	router.Handle("/api/fields", listFieldsHandler).Methods("GET")

	// start the HTTP server
	http.Handle("/", router)
	log.Printf("HTTP Server listening on %v", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}

func staticFileRouter() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	// static
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	r.Handle("/", http.RedirectHandler("/static/index.html", 302))

	return r
}

// SearchHandler can handle search requests sent over HTTP
type SearchHandler struct{}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// read the request body
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request body: %v", err), 400)
		return
	}

	//logger.Printf("request body: %s", requestBody)

	// parse the request
	var searchRequest bleve.SearchRequest
	err = json.Unmarshal(requestBody, &searchRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing query: %v", err), 400)
		return
	}

	//logger.Printf("parsed request %#v", searchRequest)

	// validate the query
	if srqv, ok := searchRequest.Query.(query.ValidatableQuery); ok {
		err = srqv.Validate()
		if err != nil {
			http.Error(w, fmt.Sprintf("error validating query: %v", err), 400)
			return
		}
	}

	// execute the query
	searchResponse, err := index.Search(&searchRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("error executing query: %v", err), 500)
		return
	}

	mailIDs := []int{}
	for _, hit := range searchResponse.Hits {
		id, _ := strconv.Atoi(hit.ID)
		mailIDs = append(mailIDs, id)
	}

	// encode the response
	mustEncode(w, mailIDs)
}

func mustEncode(w io.Writer, i interface{}) {
	if headered, ok := w.(http.ResponseWriter); ok {
		headered.Header().Set("Cache-Control", "no-cache")
		headered.Header().Set("Content-type", "application/json")
	}

	e := json.NewEncoder(w)
	if err := e.Encode(i); err != nil {
		panic(err)
	}
}
