package main

import (
	"net/smtp"
	"testing"
)

func mockSend(errToReturn error) (func(string, smtp.Auth, string, []string, []byte) error, *emailRecorder) {
	r := new(emailRecorder)
	return func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		*r = emailRecorder{addr, a, from, to, msg}
		return errToReturn
	}, r
}

type emailRecorder struct {
	addr string
	auth smtp.Auth
	from string
	to   []string
	msg  []byte
}

func TestMail_SendSuccessful(t *testing.T) {
	f, r := mockSend(nil)
	sender := &emailSender{send: f}
	body := "Hello World"
	err := sender.Send([]string{"me@example.com"}, "bob@example.com", []byte(body))

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if string(r.msg) != body {
		t.Errorf("wrong message body.expected: %s, got: %s\n", body, r.msg)
	}
}
