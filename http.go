package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"time"

	"github.com/blevesearch/bleve"
	bleveHttp "github.com/blevesearch/bleve/http"
	"github.com/blevesearch/bleve/search/query"
	"github.com/gorilla/mux"
)

type SearchHandler struct{}
type ListHandler struct{}

type ListRequest struct {
	// the mail ID to start listing from
	StartID uint64
	// the maximum number of messages to list
	Limit int
}

type SearchRequest struct {
	Query string
	// the maximum number of messages to list
	Limit     int
	Locations []string
	StartTime time.Time
	EndTime   time.Time
}

type MsgSummary struct {
	ID     string
	Header mail.Header
}

func httpServer() {
	// create a router to serve static files
	router := staticFileRouter()

	// add the API
	bleveHttp.RegisterIndexName(appName, index)
	//searchHandler := bleveHttp.NewSearchHandler(appName)
	router.Handle("/api/search", &SearchHandler{}).Methods("POST")
	router.Handle("/api/list", &ListHandler{}).Methods("POST")
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

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// read the request body
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request body: %v", err), 400)
		return
	}

	// parse the request
	var searchRequest SearchRequest
	err = json.Unmarshal(requestBody, &searchRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing request: %v", err), 400)
		return
	}

	if searchRequest.Query == "" {
		http.Error(w, fmt.Sprintf("query string cannot be empty"), 400)
		return
	}

	var bQuery query.Query
	queryQuery := query.NewQueryStringQuery(searchRequest.Query)
	if !searchRequest.StartTime.IsZero() || !searchRequest.EndTime.IsZero() {
		dateTimeQuery := query.NewDateRangeQuery(
			searchRequest.StartTime,
			searchRequest.EndTime,
		)
		bQuery = query.NewConjunctionQuery([]query.Query{queryQuery, dateTimeQuery})
	} else {
		bQuery = queryQuery
	}

	bSearchRequest := bleve.NewSearchRequest(bQuery)
	bSearchRequest.SortBy([]string{"-date"})
	bSearchRequest.Fields = []string{"Data"}

	// validate the query
	if srqv, ok := bSearchRequest.Query.(query.ValidatableQuery); ok {
		err = srqv.Validate()
		if err != nil {
			http.Error(w, fmt.Sprintf("error validating query: %v", err), 400)
			return
		}
	}

	var headers []MsgSummary
	headers, err = DoSearch(searchRequest, bSearchRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		return
	}
	mustEncode(w, headers)
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// read the request body
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request body: %v", err), 400)
		return
	}

	var searchRequest SearchRequest
	if len(requestBody) > 0 {
		err = json.Unmarshal(requestBody, &searchRequest)
		if err != nil {
			http.Error(w, fmt.Sprintf("error parsing query: %v", err), 400)
			return
		}
	}

	bQuery := bleve.NewMatchAllQuery()
	bSearchRequest := bleve.NewSearchRequest(bQuery)
	bSearchRequest.SortBy([]string{"-date"})
	bSearchRequest.Fields = []string{"Data"}

	var headers []MsgSummary
	headers, err = DoSearch(searchRequest, bSearchRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		return
	}

	// encode the response
	mustEncode(w, headers)
}

func DoSearch(hRequest SearchRequest, bRequest *bleve.SearchRequest) ([]MsgSummary, error) {

	// execute the query
	searchResult, err := index.Search(bRequest)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}

	headers := make([]MsgSummary, 0)

	for _, hit := range searchResult.Hits {
		if len(hRequest.Locations) > 0 {
			found := false
			for _, inLoc := range hRequest.Locations {
				for outLoc, _ := range hit.Locations {
					if "Data."+inLoc == outLoc {
						found = true
						goto locBreak
					}
				}
			}
		locBreak:
			if !found {
				break
			}
		}

		if hRequest.Limit > 0 && len(headers) == hRequest.Limit {
			break
		}

		fmt.Printf("hit fields %v\n", hit.Fields["Data"])
		if v, ok := hit.Fields["Data"].([]byte); ok {
			msg, err := mail.ReadMessage(bytes.NewReader(v))
			if err != nil {
				return nil, err
			}
			lr := MsgSummary{hit.ID, msg.Header}
			headers = append(headers, lr)
		} else {
			return nil, fmt.Errorf("error retrieving document")
		}
	}

	return headers, nil
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
