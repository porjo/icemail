package main

import (
	"flag"
	"fmt"

	"github.com/BurntSushi/toml"
)

const (
	smtpBindAddr  = "127.0.0.1:2525"
	httpBindAddr  = "127.0.0.1:8080"
	httpStaticDir = "static"

	appName       = "icemail"
	messageBucket = "messages"

	smtpServerAddr = "127.0.0.1:25"
)

var (
	config tomlConfig
)

type tomlConfig struct {
	SMTPBindAddr  string `toml:"smtp_bind_addr"`
	HTTPBindAddr  string `toml:"http_bind_addr"`
	HTTPStaticDir string `toml:"http_static_dir"`

	SMTPServerAddr     string `toml:"smtp_server_addr"`
	SMTPServerUsername string `toml:"smtp_server_username"`
	SMTPServerPassword string `toml:"smtp_server_password"`

	StorageDir string `toml:"storage_dir"`

	Whitelist []string `toml:"whitelist"`
}

func loadConfig() error {

	configFile := flag.String("c", "", "config filename")
	flag.Parse()

	if *configFile == "" {
		flag.PrintDefaults()
		return fmt.Errorf("Please specify a config file")
	}

	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		return fmt.Errorf("error parsing config file '%s': %s", *configFile, err)
	}

	fmt.Printf("Loaded config file '%s'\n", *configFile)

	if config.SMTPBindAddr == "" {
		config.SMTPBindAddr = smtpBindAddr
	}
	if config.HTTPBindAddr == "" {
		config.HTTPBindAddr = httpBindAddr
	}
	if config.HTTPStaticDir == "" {
		config.HTTPStaticDir = httpStaticDir
	}
	if config.SMTPServerAddr == "" {
		config.SMTPServerAddr = smtpServerAddr
	}

	return nil
}
