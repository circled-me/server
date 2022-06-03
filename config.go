package main

import "os"

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
