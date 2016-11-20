# Icemail

Email catcher for development environments. Solves the problem of test emails becoming a nuisance to developers and customers.

- acts as a 'smart relay' which by default stores all incoming mail rather than relaying it.
- recipient addresses/domains can be whitelisted for auto-delivery, otherwise mail can be manually released via a web interface
- single binary which does everything: mail server, mail client, web server
- easy to install and configure

## Credits

- Inspired by [MailHog](https://github.com/mailhog/MailHog/) which in turn was inspired by [MailCatcher](http://mailcatcher.me/)
- Uses: [Bleve](http://www.blevesearch.com) full-text searching, [BoltDB](https://github.com/boltdb/bolt) key/value database

