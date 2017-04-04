# Icemail

[![Build Status](https://travis-ci.org/porjo/icemail.svg?branch=master)](https://travis-ci.org/porjo/icemail)
[![Coverage Status](https://coveralls.io/repos/github/porjo/icemail/badge.svg?branch=master)](https://coveralls.io/github/porjo/icemail?branch=master)

**Work in progress**

Email catcher for development environments. Solves the problem of test emails becoming a nuisance to developers and customers.

- acts as a 'smart relay' which by default stores all incoming mail rather than relaying it.
- recipient addresses/domains can be whitelisted for auto-delivery, otherwise mail can be manually released via a web interface
- single binary which does everything: mail server, mail client, web server
- easy to install and configure

## Setup

- download tar file from [releases](https://github.com/porjo/icemail/releases) page and unpack somewhere convenient
- edit `config.toml` to suit
- run with `./icemail -c config.toml`
- point browser at `http://localhost:8080`

By default, icemail will:

- listen on port `:2525` for SMTP connections
- listen on port `:8080` for HTTP client connections
- forward outbound email to localhost on port `:25`

## Credits

- Inspired by [MailHog](https://github.com/mailhog/MailHog/) which in turn was inspired by [MailCatcher](http://mailcatcher.me/)
- Uses: [Bleve](http://www.blevesearch.com) full-text searching, [BoltDB](https://github.com/boltdb/bolt) key/value database

