package main

import (
	"fmt"
	"log"
	"net/smtp"
	"os"

	"github.com/blevesearch/bleve"
	"github.com/mhale/smtpd"
)

func main() {
	var err error

	if err = loadConfig(); err != nil {
		log.Fatal(err)
	}

	var indexDir string
	if config.StorageDir != "" {
		indexDir = config.StorageDir + "/" + appName + ".db"
	} else {
		indexDir = appName + ".db"
	}

	// try opening index, otherwise try creating new
	index, err = bleve.Open(indexDir)
	if err != nil {
		// if the index exists but couldn't be opened, don't proceed
		if _, err2 := os.Stat(indexDir); err2 == nil {
			log.Fatalf("Error opening index '%s': %s\n", indexDir, err)
		}

		fmt.Printf("Creating database '%s'\n", indexDir)

		mapping := buildIndexMapping()
		index, err = bleve.New(indexDir, mapping)
		if err != nil {
			log.Fatalf("Error creating index '%s': %s\n", indexDir, err)
		}
	} else {
		fmt.Printf("Loading database '%s'\n", indexDir)
	}

	// sanity check
	if config.SMTPServerAddr == config.SMTPBindAddr {
		log.Fatal("SMTP server and bind address cannot be the same!")
	}

	mailConfig := MailConfig{
		Username:   config.SMTPServerUsername,
		Password:   config.SMTPServerPassword,
		ServerAddr: config.SMTPServerAddr,
	}
	if mailSender, err = NewEmailSender(mailConfig); err != nil {
		log.Fatalf("Error configuring email settings: %s\n", err)
	}

	//Test SMTP server connection
	var c *smtp.Client
	if c, err = smtp.Dial(config.SMTPServerAddr); err != nil {
		log.Fatalf("Error connecting to SMTP server '%s': %s\n", config.SMTPServerAddr, err)
	}
	c.Close()

	//go outputStats()
	go httpServer()

	fmt.Printf("SMTP server listening on %s\n", config.SMTPBindAddr)
	smtpd.ListenAndServe(config.SMTPBindAddr, HandleMessage, appName, "")
}

/*
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
	}
}
*/
