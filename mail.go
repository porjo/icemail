package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

type mailHandler func(net.Addr, string, []string, []byte) error

func (fn mailHandler) HandleMessage(origin net.Addr, from string, to []string, data []byte) {
	if err := fn(origin, from, to, data); err != nil {
		log.Println(err)
	}
}

func handleMessage(origin net.Addr, from string, to []string, data []byte) error {
	var err error

	var msg *mail.Message
	msg, err = mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return err
	}

	doc := bleveDoc{Type: "message", Header: msg.Header, Data: string(data)}

	id := fmt.Sprintf("%v", time.Now().UnixNano())
	if err := index.Index(id, doc); err != nil {
		return err
	}

	subject := msg.Header.Get("Subject")
	log.Printf("Received mail ID %s, To: '%s', From: '%s', Subject: '%s'\n", id, to[0], from, subject)
	return nil
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func sendMail(hRequest SearchRequest, docID string) (MailResult, error) {
	docQuery := query.NewDocIDQuery([]string{docID})

	bRequest := bleve.NewSearchRequest(docQuery)
	bRequest.Fields = []string{"Data"}

	var hResult MailResult
	searchResult, err := index.Search(bRequest)
	if err != nil {
		return hResult, fmt.Errorf("error executing query: %v", err)
	}

	var raw string
	var delivered time.Time
	var msg *mail.Message
	if len(searchResult.Hits) == 1 {
		hit := searchResult.Hits[0]
		var ok bool
		if raw, ok = hit.Fields["Data"].(string); ok {
			msg, err = mail.ReadMessage(strings.NewReader(raw))
			if err != nil {
				return hResult, err
			}
		} else {
			return hResult, fmt.Errorf("error retrieving document")
		}

		to := msg.Header["To"]
		from := msg.Header.Get("From")

		var host string
		if host, _, err = net.SplitHostPort(config.SMTPServerAddr); err != nil {
			return hResult, err
		}
		var auth smtp.Auth
		if config.SMTPServerUsername != "" && config.SMTPServerPassword != "" {
			auth = smtp.PlainAuth("", config.SMTPServerUsername, config.SMTPServerPassword, host)
		}
		if err = smtp.SendMail(config.SMTPServerAddr, auth, from, to, []byte(raw)); err != nil {
			return hResult, fmt.Errorf("error sending mail with ID %s: %v", docID, err)
		} else {
			subject := msg.Header.Get("Subject")
			log.Printf("Sending mail ID %s, To: '%s', Subject: '%s'\n", docID, to[0], subject)
		}
		delivered = time.Now()
	} else {
		return hResult, fmt.Errorf("mail with ID %s not found", docID)
	}

	// Remove document from index, then re-add with 'delivered' time set
	if !delivered.IsZero() {

		if err = index.Delete(docID); err != nil {
			return hResult, err
		}
		doc := bleveDoc{Type: "message", Header: msg.Header, Data: raw, Delivered: delivered}
		if err := index.Index(docID, doc); err != nil {
			return hResult, err
		}
		hResult.Success = true
	}

	return hResult, nil
}
