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
	"strconv"

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
	var searchRequest bleve.SearchRequest
	err = json.Unmarshal(requestBody, &searchRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing query: %v", err), 400)
		return
	}

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

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// read the request body
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading request body: %v", err), 400)
		return
	}

	var listRequest ListRequest
	err = json.Unmarshal(requestBody, &listRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing query: %v", err), 400)
		return
	}

	headers := make(map[uint64]mail.Header)
	err = db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(messageBucket))

		addHeader := func(headers map[uint64]mail.Header, k, v []byte) error {
			idx := binary.BigEndian.Uint64(k)
			msg, err := mail.ReadMessage(bytes.NewReader(v))
			if err != nil {
				return err
			}
			headers[idx] = msg.Header
			return nil
		}

		c := b.Cursor()
		if listRequest.StartID > 0 {
			startID := itob(listRequest.StartID)
			for k, v := c.Seek(startID); k != nil; k, v = c.Next() {
				if listRequest.Limit > 0 && len(headers) == listRequest.Limit {
					break
				}
				if err = addHeader(headers, k, v); err != nil {
					return err
				}
			}
		} else {
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if listRequest.Limit > 0 && len(headers) == listRequest.Limit {
					break
				}
				if err = addHeader(headers, k, v); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error listing messages: %v", err), 400)
		return
	}

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
