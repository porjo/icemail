package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/blevesearch/bleve"
	bleveHttp "github.com/blevesearch/bleve/http"
	"github.com/blevesearch/bleve/search/query"
	"github.com/gorilla/mux"
)

// hard limit on number of results returned
const ResultLimit = 200

const SearchPrefixLen = 3
const SearchFuzziness = 2

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
	ID        string
	Header    mail.Header
	Body      string
	Delivered *time.Time `json:"Delivered,omitempty"`
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
	fmt.Printf("HTTP Server listening on %v\n", config.HTTPBindAddr)
	log.Fatal(http.ListenAndServe(config.HTTPBindAddr, nil))
}

func staticFileRouter() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	// static
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(config.HTTPStaticDir))))

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
		locQueries := make([]query.Query, 0)

		if len(searchRequest.Locations) == 0 {
			searchRequest.Locations = append(searchRequest.Locations, "")
		}

		for _, location := range searchRequest.Locations {

			if strings.Contains(searchRequest.Query, " ") {
				tmpQuery := query.NewMatchPhraseQuery(searchRequest.Query)
				if location != "" {
					tmpQuery.SetField(locationsBase + location)
				}
				matchQuery = tmpQuery
			} else {
				if utf8.RuneCountInString(searchRequest.Query) <= SearchPrefixLen {
					http.Error(w, fmt.Sprintf("query string too short"), 400)
					return
				}
				tmpQuery := query.NewMatchQuery(searchRequest.Query)
				tmpQuery.SetFuzziness(SearchFuzziness)
				tmpQuery.SetPrefix(SearchPrefixLen)
				if location != "" {
					tmpQuery.SetField(locationsBase + location)
				}
				matchQuery = tmpQuery
			}

			locQueries = append(locQueries, matchQuery)
		}
		matchQuery = query.NewDisjunctionQuery(locQueries)
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
	bSearchRequest.Fields = []string{"Data", "Delivered"}
	bSearchRequest.From = searchRequest.Offset

	switch {
	case searchRequest.Limit > ResultLimit:
		http.Error(w, fmt.Sprintf("request limit of %d is greater than max allowed limit of %d", searchRequest.Limit, ResultLimit), 400)
		return
	case searchRequest.Limit > 0:
		bSearchRequest.Size = searchRequest.Limit
	default:
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
	bSearchRequest.Fields = []string{"Data", "Delivered"}

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
	var httpStatus int
	docID := mux.Vars(req)["docID"]

	httpStatus, err = sendMailDoc(docID)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), httpStatus)
		return
	}

	result := MailResult{}
	result.Success = true

	mustEncode(w, result)
}

func doSearch(hRequest SearchRequest, bRequest *bleve.SearchRequest, includeBody bool) (SearchResult, error) {
	var hResult SearchResult
	searchResult, err := index.Search(bRequest)
	if err != nil {
		return hResult, fmt.Errorf("error executing query: %v", err)
	}

	emails := make([]Email, 0)
	for _, hit := range searchResult.Hits {

		if hRequest.Limit > 0 && len(emails) == hRequest.Limit {
			break
		}

		if v, ok := hit.Fields["Data"].(string); ok {
			msg, err := mail.ReadMessage(strings.NewReader(v))
			if err != nil {
				return hResult, err
			}
			lr := Email{ID: hit.ID, Header: msg.Header}

			if includeBody {
				lr.Body, err = getBody(msg)
				if err != nil {
					return hResult, err
				}
			}

			if deliveredS, ok := hit.Fields["Delivered"].(string); ok {
				var d time.Time
				if d, err = time.Parse(time.RFC3339, deliveredS); err != nil {
					return hResult, err
				}
				lr.Delivered = &d
			}

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

func getBody(msg *mail.Message) (string, error) {
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(msg.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}

			if p.FileName() == "" {
				b, err := ioutil.ReadAll(p)
				if err != nil {
					return "", err
				}
				return string(b), nil
			}
		}
	} else if mediaType == "text/plain" && msg.Header.Get("Content-Transfer-Encoding") == "quoted-printable" {
		b, err := ioutil.ReadAll(quotedprintable.NewReader(msg.Body))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", nil
}
