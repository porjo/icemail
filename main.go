package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/boltdb/bolt"
	"github.com/porjo/icemail/smtpd"
)

var db *bolt.DB
var mailAddr string = "127.0.0.1:2525"
var httpAddr string = "127.0.0.1:8080"
var index bleve.Index
var appName string = "icemail"
var staticPath string = "static"

func main() {
	var err error
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err = bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// try opening index, otherwise try creating new
	index, err = bleve.Open(appName + ".bleve")
	if err != nil {
		log.Printf("Creating new search index...")
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(appName+".bleve", mapping)
		if err != nil {
			log.Fatal(err)
		}
	}

	go outputStats()
	go httpServer()

	log.Printf("Mail server listening on %s...\n", mailAddr)
	smtpd.ListenAndServe(mailAddr, handler(mailHandler), appName, "")
}

func outputStats() {
	for {
		// Wait for 10s.
		time.Sleep(10 * time.Second)

		// Grab the current stats and diff them.
		stats := db.Stats()
		// Encode stats to JSON and print to STDERR.
		json.NewEncoder(os.Stderr).Encode(stats)

		if err := db.View(func(tx *bolt.Tx) error {
			buckets := 0
			keys := 0
			err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
				buckets++
				c := b.Cursor()
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					keys++
				}
				return nil
			})
			if err != nil {
				return err
			}
			log.Printf("buckets %d keys %d\n", buckets, keys)
			return nil
		}); err != nil {
			log.Printf("bolt view err %s\n", err)
		}

		/*
			// search for some text
			query := bleve.NewMatchQuery("pace7")
			search := bleve.NewSearchRequest(query)
			log.Printf("index %v\n", index)
			searchResults, err := index.Search(search)
			if err != nil {
				log.Printf("index search err %s\n", err)
			}
			log.Printf("results %s\n", searchResults.Hits)
		*/
	}
}
