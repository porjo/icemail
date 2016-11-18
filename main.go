package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/mail"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/porjo/icemail/smtpd"
)

var db *bolt.DB
var addr string = "127.0.0.1:2525"

type handler func(net.Addr, string, []string, []byte) error

func main() {
	var err error
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	db, err = bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	go func() {

		for {
			// Wait for 10s.
			time.Sleep(10 * time.Second)

			// Grab the current stats and diff them.
			stats := db.Stats()
			// Encode stats to JSON and print to STDERR.
			json.NewEncoder(os.Stderr).Encode(stats)

			db.View(func(tx *bolt.Tx) error {
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
			})
		}
	}()

	log.Printf("Listening on %s...\n", addr)
	smtpd.ListenAndServe(addr, handler(mailHandler), "MyServerApp", "")

}

func (fn handler) HandleMessage(origin net.Addr, from string, to []string, data []byte) {

	if err := fn(origin, from, to, data); err != nil {
		log.Println(err)
	}

}

func mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	var err error
	// Execute several commands within a read-write transaction.
	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(to[0]))
		if err != nil {
			return err
		}
		if err := b.Put([]byte(time.Now().Format(time.RFC3339)), data); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	var msg *mail.Message
	msg, err = mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return err
	}
	subject := msg.Header.Get("Subject")
	log.Printf("Received mail from %s for %s with subject %s", from, to[0], subject)

	return nil
}
