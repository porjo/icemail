package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"testing"
	"time"

	"github.com/blevesearch/bleve"
)

const emailStr = `Date: Tue, 04 Apr 2017 19:02:05 +1000
From: from@example.com
To: to@example.com
Subject: test subject
Cc: cc@example.com

test message`

type emailRecorder struct {
	addr string
	auth smtp.Auth
	from string
	to   []string
	msg  []byte
}

func TestMain(m *testing.M) {
	err := setupTest()
	if err != nil {
		log.Fatalln(err)
	}
	retCode := m.Run()
	//   myTeardownFunction()
	os.Exit(retCode)
}

func TestHandle(t *testing.T) {
	origin, err := net.ResolveIPAddr("ip", "192.168.1.1")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	err = handleMessage(origin, "from@example.com", []string{"to@example.com"}, []byte(emailStr))
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestSendMail(t *testing.T) {
	msg, err := mail.ReadMessage(bytes.NewReader([]byte(emailStr)))
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	rcpts := []string{"to@example.com"}
	doc := bleveDoc{Type: "message", Header: msg.Header, Data: emailStr, Recipients: rcpts}

	id := fmt.Sprintf("%v", time.Now().UnixNano())
	if err := index.Index(id, doc); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	_, err = sendMailDoc(id)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func setupTest() error {
	var err error
	mapping := buildIndexMapping()
	index, err = bleve.NewMemOnly(mapping)
	if err != nil {
		return err
	}
	f, _ := mockSend(nil)
	mailSender = &emailSender{send: f}
	return nil
}

func mockSend(errToReturn error) (func(string, smtp.Auth, string, []string, []byte) error, *emailRecorder) {
	r := new(emailRecorder)
	return func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		*r = emailRecorder{addr, a, from, to, msg}
		return errToReturn
	}, r
}
