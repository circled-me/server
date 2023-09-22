package main

import "os"

var (
	TLS_DOMAINS = ""                        // TODO make this env variable, e.g. "example.com example2.com"
	DEBUG_MODE  = true                      // TODO make this env variable
	PUSH_SERVER = "http://192.168.1.6:8081" // TODO: env var
)

func GetMySQLDSN() string {
	dsn := os.Getenv("CIRCLED_MYSQL_DSN")
	if dsn == "" {
		// TODO: remove
		dsn = "root:@tcp(127.0.0.1:3306)/circled?charset=utf8mb4&parseTime=True&loc=Local"
	}
	return dsn
}

func GetBindAddress() string {
	bind := os.Getenv("CIRCLED_BIND_ADDRESS")
	if bind == "" {
		bind = "0.0.0.0:8080"
	}
	return bind
}
