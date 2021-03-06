package main

import (
	"bytes"
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

var mailSender EmailSender

type EmailSender interface {
	Send(to []string, from string, body []byte) error
}

type emailSender struct {
	conf MailConfig
	send func(string, smtp.Auth, string, []string, []byte) error
}

type MailConfig struct {
	Username   string
	Password   string
	ServerAddr string

	auth smtp.Auth
}

func NewEmailSender(conf MailConfig) (EmailSender, error) {
	var host string
	var err error
	if host, _, err = net.SplitHostPort(config.SMTPServerAddr); err != nil {
		return nil, fmt.Errorf("error parsing SMTPServerAddr: %s", err)
	}
	if conf.Username != "" && conf.Password != "" {
		conf.auth = smtp.PlainAuth("", conf.Username, conf.Password, host)
	}
	return &emailSender{conf, smtp.SendMail}, nil
}

func HandleMessage(origin net.Addr, from string, to []string, data []byte) {
	if err := handleMessage(origin, from, to, data); err != nil {
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

	subject := msg.Header.Get("Subject")

	var addresses []*mail.Address
	var delivered time.Time
	if addresses, err = msg.Header.AddressList("To"); err == nil {
		if isWhitelisted(addresses) {
			log.Printf("Email whitelisted, To: '%s', From: '%s', Subject: '%s'\n", to[0], from, subject)
			err = sendMail(data, *msg, to)
			if err != nil {
				return err
			}
			delivered = time.Now()
		}
	}

	// Make sure header field has a valid date so it can be expired later
	// FIXME: why doesn't this modification survive the save!?
	if msg.Header.Get("Date") == "" {
		now := time.Now().Format(RFC1123ZnoPadDay)
		msg.Header["Date"] = []string{now}
	}

	fmt.Printf("handle message to %+v\n", to)
	doc := bleveDoc{Type: "message", Header: msg.Header, Data: string(data), Delivered: delivered, Recipients: to}

	id := fmt.Sprintf("%v", time.Now().UnixNano())
	if err := index.Index(id, doc); err != nil {
		return err
	}

	log.Printf("Received mail ID %s, To: '%s', From: '%s', Subject: '%s'\n", id, to[0], from, subject)
	return nil
}

/*
// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
*/

func sendMailDoc(docID string) (int, error) {
	docQuery := query.NewDocIDQuery([]string{docID})

	bRequest := bleve.NewSearchRequest(docQuery)
	bRequest.Fields = []string{"Data", "Delivered", "Recipients"}

	searchResult, err := index.Search(bRequest)
	if err != nil {
		return 500, fmt.Errorf("error executing query: %v", err)
	}

	var msg *mail.Message
	if len(searchResult.Hits) != 1 {
		return 404, fmt.Errorf("mail with ID %s not found", docID)
	}

	var raw string
	var ok bool
	hit := searchResult.Hits[0]
	if _, ok = hit.Fields["Delivered"].(string); ok {
		return 400, fmt.Errorf("mail with ID %s already delivered", docID)
	}

	if raw, ok = hit.Fields["Data"].(string); ok {
		msg, err = mail.ReadMessage(strings.NewReader(raw))
		if err != nil {
			return 500, err
		}
	} else {
		return 500, fmt.Errorf("error retrieving document")
	}

	var recipients []string
	if rcpts, ok := hit.Fields["Recipients"].([]interface{}); ok {
		recipients = make([]string, 0)
		for _, r := range rcpts {
			if rcpt, ok := r.(string); ok {
				recipients = append(recipients, rcpt)
			}
		}
	} else {
		// Bleve doesn't handle arrays properly.
		// A string slice with a single element will be returned as a string.
		// See: https://github.com/blevesearch/bleve/issues/570
		if rcpt, ok := hit.Fields["Recipients"].(string); ok {
			recipients = []string{rcpt}
		} else {
			return 400, fmt.Errorf("mail with ID %s has no recipients", docID)
		}
	}

	if err = sendMail([]byte(raw), *msg, recipients); err != nil {
		return 500, fmt.Errorf("error sending mail with ID %s: %v", docID, err)
	}

	delivered := time.Now()

	if err = index.Delete(docID); err != nil {
		return 500, err
	}
	doc := bleveDoc{Type: "message", Header: msg.Header, Data: raw, Delivered: delivered, Recipients: recipients}
	if err := index.Index(docID, doc); err != nil {
		return 500, err
	}

	return 200, nil
}

func sendMail(data []byte, msg mail.Message, rcpts []string) error {
	var err error
	from := msg.Header.Get("From")

	if err = mailSender.Send(rcpts, from, data); err != nil {
		return err
	} else {
		subject := msg.Header.Get("Subject")
		log.Printf("Sending mail, Recipients: %s, From: '%s', Subject: '%s'\n", rcpts, from, subject)
	}

	return nil
}

func isWhitelisted(emails []*mail.Address) bool {
	for _, e := range emails {
		for _, w := range config.Whitelist {
			if strings.Contains(w, "@") {
				if w == e.Address {
					return true
				}
			} else {
				parts := strings.Split(e.Address, "@")
				if len(parts) == 2 && w == parts[1] {
					return true
				}
			}
		}
	}
	return false
}

func (e *emailSender) Send(to []string, from string, body []byte) error {
	return e.send(e.conf.ServerAddr, e.conf.auth, from, to, body)
}
