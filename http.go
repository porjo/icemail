package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/blevesearch/bleve"
	bleveHttp "github.com/blevesearch/bleve/http"
	"github.com/blevesearch/bleve/search/query"
	"github.com/gorilla/mux"
)

type SearchDocHandler struct{}
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

type EmailResult struct {
	ID     string
	Header mail.Header
	Body   string
}

func httpServer() {
	// create a router to serve static files
	router := staticFileRouter()

	// add the API
	bleveHttp.RegisterIndexName(appName, index)
	//searchHandler := bleveHttp.NewSearchHandler(appName)
	router.Handle("/api/search", &SearchHandler{}).Methods("POST")
	router.Handle("/api/search/{DocID}", &SearchDocHandler{}).Methods("GET")
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
	matchQuery := query.NewMatchQuery(searchRequest.Query)
	if !searchRequest.StartTime.IsZero() || !searchRequest.EndTime.IsZero() {
		dateTimeQuery := query.NewDateRangeQuery(
			searchRequest.StartTime,
			searchRequest.EndTime,
		)
		bQuery = query.NewConjunctionQuery([]query.Query{matchQuery, dateTimeQuery})
	} else {
		bQuery = matchQuery
	}

	bSearchRequest := bleve.NewSearchRequest(bQuery)
	bSearchRequest.SortBy([]string{"-Header.Date"})
	bSearchRequest.Fields = []string{"Data"}

	// validate the query
	if srqv, ok := bSearchRequest.Query.(query.ValidatableQuery); ok {
		err = srqv.Validate()
		if err != nil {
			http.Error(w, fmt.Sprintf("error validating query: %v", err), 400)
			return
		}
	}

	var headers []EmailResult
	headers, err = DoSearch(searchRequest, bSearchRequest, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		return
	}
	mustEncode(w, headers)
}

func (h *SearchDocHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	vars := mux.Vars(req)
	docID := vars["DocID"]

	docQuery := query.NewDocIDQuery([]string{docID})

	bSearchRequest := bleve.NewSearchRequest(docQuery)
	bSearchRequest.Fields = []string{"Data"}

	var headers []EmailResult
	headers, err = DoSearch(SearchRequest{}, bSearchRequest, true)
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
	bSearchRequest.SortBy([]string{"-Header.Date"})
	bSearchRequest.Fields = []string{"Data"}

	var headers []EmailResult
	headers, err = DoSearch(searchRequest, bSearchRequest, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		return
	}

	// encode the response
	mustEncode(w, headers)
}

func DoSearch(hRequest SearchRequest, bRequest *bleve.SearchRequest, includeBody bool) ([]EmailResult, error) {
	searchResult, err := index.Search(bRequest)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}

	headers := make([]EmailResult, 0)

	for _, hit := range searchResult.Hits {
		if len(hRequest.Locations) > 0 {
			found := false
			for _, inLoc := range hRequest.Locations {
				for outLoc, _ := range hit.Locations {
					if locationsBase+inLoc == outLoc {
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

		if v, ok := hit.Fields["Data"].(string); ok {
			msg, err := mail.ReadMessage(strings.NewReader(v))
			if err != nil {
				return nil, err
			}
			body := ""
			if includeBody {
				body = v
			}
			lr := EmailResult{hit.ID, msg.Header, body}
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
