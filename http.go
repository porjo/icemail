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

const ResultLimit = 100

type MailHandler struct{}
type SearchDocHandler struct{}
type SearchHandler struct{}

type SearchRequest struct {
	Query string
	// the mail ID to start listing from
	//StartID uint64
	// the maximum number of messages to list
	Limit     int
	Offset    int
	Locations []string
	StartTime time.Time
	EndTime   time.Time
}

type SearchResult struct {
	Total  uint64
	Offset int
	Emails []Email
}

type MailResult struct {
	Success bool
}

type Email struct {
	ID     string
	Header mail.Header
	Body   string `json:"Body,omitempty"`
}

func httpServer() {
	// create a router to serve static files
	router := staticFileRouter()

	// add the API
	bleveHttp.RegisterIndexName(appName, index)
	router.Handle("/api/search", &SearchHandler{}).Methods("POST")
	router.Handle("/api/search/{docID}", &SearchDocHandler{}).Methods("GET")
	router.Handle("/api/mail/{docID}", &MailHandler{}).Methods("GET")
	router.Handle("/api/list", &SearchHandler{}).Methods("POST")
	listFieldsHandler := bleveHttp.NewListFieldsHandler(appName)
	router.Handle("/api/fields", listFieldsHandler).Methods("GET")
	listIndexesHandler := bleveHttp.NewListIndexesHandler()
	router.Handle("/api/indexes", listIndexesHandler).Methods("GET")
	debugHandler := bleveHttp.NewDebugDocumentHandler(appName)
	debugHandler.DocIDLookup = func(req *http.Request) string {
		return mux.Vars(req)["docID"]
	}
	router.Handle("/api/debug/{docID}", debugHandler).Methods("GET")

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

	var bQuery query.Query
	var matchQuery query.Query

	if searchRequest.Query == "" {
		matchQuery = bleve.NewMatchAllQuery()
	} else {
		matchQuery = query.NewMatchQuery(searchRequest.Query)
	}
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
	bSearchRequest.From = searchRequest.Offset
	if searchRequest.Limit > 0 && searchRequest.Limit <= ResultLimit {
		bSearchRequest.Size = searchRequest.Limit
	} else {
		bSearchRequest.Size = ResultLimit
	}

	// validate the query
	if srqv, ok := bSearchRequest.Query.(query.ValidatableQuery); ok {
		err = srqv.Validate()
		if err != nil {
			http.Error(w, fmt.Sprintf("error validating query: %v", err), 400)
			return
		}
	}

	var result SearchResult
	result, err = doSearch(searchRequest, bSearchRequest, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		return
	}

	result.Offset = searchRequest.Offset
	mustEncode(w, result)
}

func (h *SearchDocHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	docID := mux.Vars(req)["docID"]

	docQuery := query.NewDocIDQuery([]string{docID})

	bSearchRequest := bleve.NewSearchRequest(docQuery)
	bSearchRequest.Fields = []string{"Data"}

	var result SearchResult
	result, err = doSearch(SearchRequest{}, bSearchRequest, true)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		return
	}
	mustEncode(w, result)
}

func (h *MailHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	var searchRequest SearchRequest
	docID := mux.Vars(req)["docID"]

	/*
		// read the request body
		requestBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading request body: %v", err), 400)
			return
		}

		// parse the request
		err = json.Unmarshal(requestBody, &searchRequest)
		if err != nil {
			http.Error(w, fmt.Sprintf("error parsing request: %v", err), 400)
			return
		}
	*/

	var result MailResult
	result, err = sendMail(searchRequest, docID)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		return
	}
	mustEncode(w, result)
}

func doSearch(hRequest SearchRequest, bRequest *bleve.SearchRequest, includeBody bool) (SearchResult, error) {
	var hResult SearchResult
	searchResult, err := index.Search(bRequest)
	if err != nil {
		return hResult, fmt.Errorf("error executing query: %v", err)
	}

	emails := make([]Email, 0)

	fmt.Printf("hits %d\n", len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		if len(hRequest.Locations) > 0 && len(hit.Locations) > 0 {
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

		if hRequest.Limit > 0 && len(emails) == hRequest.Limit {
			break
		}

		if v, ok := hit.Fields["Data"].(string); ok {
			msg, err := mail.ReadMessage(strings.NewReader(v))
			if err != nil {
				return hResult, err
			}
			body := ""
			if includeBody {
				body = v
			}
			lr := Email{hit.ID, msg.Header, body}
			emails = append(emails, lr)
		} else {
			return hResult, fmt.Errorf("error retrieving document")
		}
	}

	hResult.Total = searchResult.Total
	hResult.Emails = emails
	return hResult, nil
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
