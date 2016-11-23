package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"net/mail"
	"strconv"

	"github.com/boltdb/bolt"
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

	// Execute several commands within a read-write transaction.
	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(messageBucket))
		if err != nil {
			return err
		}
		id, _ := b.NextSequence()

		if err := b.Put(itob(id), data); err != nil {
			return err
		}

		if err := index.Index(strconv.FormatUint(id, 10), msg.Header); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	subject := msg.Header.Get("Subject")
	log.Printf("Received mail from %s for %s with subject %s", from, to[0], subject)

	return nil
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
