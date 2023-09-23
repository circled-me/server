package config

import (
	"os"
	"strings"
)

var (
	TLS_DOMAINS  = "" // e.g. "example.com example2.com"
	PUSH_SERVER  = "http://192.168.1.6:8081"
	MYSQL_DSN    = "root:@tcp(127.0.0.1:3306)/circled?charset=utf8mb4&parseTime=True&loc=Local"
	BIND_ADDRESS = "0.0.0.0:8080"
	DEBUG_MODE   = true
)

func init() {
	readEnvString("TLS_DOMAINS", &TLS_DOMAINS)
	readEnvString("PUSH_SERVER", &PUSH_SERVER)
	readEnvString("MYSQL_DSN", &MYSQL_DSN)
	readEnvString("BIND_ADDRESS", &BIND_ADDRESS)
	readEnvBool("DEBUG_MODE", &DEBUG_MODE)
}

func readEnvString(name string, value *string) {
	v := os.Getenv(name)
	if v == "" {
		return
	}
	*value = v
}

func readEnvBool(name string, value *bool) {
	v := strings.ToLower(os.Getenv(name))
	if v == "true" || v == "1" || v == "yes" || v == "on" {
		*value = true
	} else if v == "false" || v == "0" || v == "no" || v == "off" {
		*value = false
	}
}
