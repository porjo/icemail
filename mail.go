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

	/*
		if msg.Header.Get("Date") == "" {
			now := time.Now().Format(RFC1123ZnoPadDay)
			msg.Header["Date"] = []string{now}
		}
		if msg.Header.Get("To") == "" {
			msg.Header["To"] = to
		}
		if msg.Header.Get("From") == "" {
			msg.Header["From"] = []string{from}
		}
	*/

	doc := bleveDoc{"message", msg.Header, string(data)}

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

	if len(searchResult.Hits) == 1 {
		hit := searchResult.Hits[0]
		var raw string
		var ok bool
		var msg *mail.Message
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
		err = smtp.SendMail(smtpAddr, nil, from, to, []byte(raw))
		if err != nil {
			return hResult, fmt.Errorf("error sending mail with ID %s: %v", docID, err)
		} else {
			subject := msg.Header.Get("Subject")
			log.Printf("Sending mail ID %s, To: '%s', Subject: '%s'\n", docID, to[0], subject)
		}
		hResult.Success = true
	} else {
		return hResult, fmt.Errorf("mail with ID %s not found", docID)
	}

	return hResult, nil
}
