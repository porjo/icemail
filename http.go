package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"time"

	"github.com/blevesearch/bleve"
	bleveHttp "github.com/blevesearch/bleve/http"
	"github.com/blevesearch/bleve/search/query"
	"github.com/boltdb/bolt"
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
	Days      int
}

type MsgSummary struct {
	ID     uint64
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
		http.Error(w, fmt.Sprintf("error parsing query: %v", err), 400)
		return
	}

	bSearchRequest := bleve.NewSearchRequest(query.NewQueryStringQuery(searchRequest.Query))
	// validate the query
	if srqv, ok := bSearchRequest.Query.(query.ValidatableQuery); ok {
		err = srqv.Validate()
		if err != nil {
			http.Error(w, fmt.Sprintf("error validating query: %v", err), 400)
			return
		}
	}

	// execute the query
	searchResult, err := index.Search(bSearchRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("error executing query: %v", err), 500)
		return
	}

	headers := make(TimeSlice, 0)
	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(messageBucket))

		for _, hit := range searchResult.Hits {
			if len(searchRequest.Locations) > 0 {
				found := false
				for _, inLoc := range searchRequest.Locations {
					for outLoc, _ := range hit.Locations {
						if inLoc == outLoc {
							found = true
							goto locBreak
						}
					}
				}
			locBreak:
				if !found {
					return nil
				}
			}

			idx, err := strconv.ParseUint(hit.ID, 10, 64)
			v := b.Get(itob(idx))
			if searchRequest.Limit > 0 && len(headers) == searchRequest.Limit {
				break
			}
			msg, err := mail.ReadMessage(bytes.NewReader(v))
			if err != nil {
				return err
			}
			lr := MsgSummary{idx, msg.Header}
			headers = append(headers, lr)
		}
		return nil
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error listing messages: %v", err), 500)
		return
	}

	sort.Sort(headers)
	// encode the response
	mustEncode(w, headers)
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// read the request body
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request body: %v", err), 400)
		return
	}

	var listRequest ListRequest
	if len(requestBody) > 0 {
		err = json.Unmarshal(requestBody, &listRequest)
		if err != nil {
			http.Error(w, fmt.Sprintf("error parsing query: %v", err), 400)
			return
		}
	}

	headers := make(TimeSlice, 0)
	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(messageBucket))

		c := b.Cursor()
		if listRequest.StartID > 0 {
			startID := itob(listRequest.StartID)
			for k, v := c.Seek(startID); k != nil; k, v = c.Next() {
				if listRequest.Limit > 0 && len(headers) == listRequest.Limit {
					break
				}
				msg, err := mail.ReadMessage(bytes.NewReader(v))
				if err != nil {
					return err
				}
				idx := binary.BigEndian.Uint64(k)
				lr := MsgSummary{idx, msg.Header}
				headers = append(headers, lr)
			}
		} else {
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if listRequest.Limit > 0 && len(headers) == listRequest.Limit {
					break
				}
				msg, err := mail.ReadMessage(bytes.NewReader(v))
				if err != nil {
					return err
				}
				idx := binary.BigEndian.Uint64(k)
				lr := MsgSummary{idx, msg.Header}
				headers = append(headers, lr)
			}
		}
		return nil
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error listing messages: %v", err), 500)
		return
	}

	sort.Sort(headers)

	// encode the response
	mustEncode(w, headers)
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

type TimeSlice []MsgSummary

func (s TimeSlice) Len() int {
	return len(s)
}

func (s TimeSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s TimeSlice) Less(i, j int) bool {
	iDate, _ := time.Parse(time.RFC1123Z, s[i].Header.Get("Date"))
	jDate, _ := time.Parse(time.RFC1123Z, s[j].Header.Get("Date"))

	return iDate.After(jDate)
}
