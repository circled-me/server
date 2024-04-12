package config

import (
	"os"
	"strings"
)

var (
	TLS_DOMAINS        = ""                    // e.g. "example.com,example2.com"
	DEFAULT_ASSET_PATH = "<year>/<month>/<id>" // also available: <name>, <Month>
	PUSH_SERVER        = "https://push.circled.me"
	MYSQL_DSN          = "" // MySQL will be used if this is set
	SQLITE_FILE        = "" // SQLite will be used if MYSQL_DSN is not configured and this is set
	BIND_ADDRESS       = "0.0.0.0:8080"
	TMP_DIR            = "/tmp" // Used for temporary video conversion, etc (in case of S3 bucket)
	DEBUG_MODE         = true
)

func init() {
	readEnvString("TLS_DOMAINS", &TLS_DOMAINS)
	readEnvString("PUSH_SERVER", &PUSH_SERVER)
	readEnvString("MYSQL_DSN", &MYSQL_DSN)
	readEnvString("SQLITE_FILE", &SQLITE_FILE)
	readEnvString("BIND_ADDRESS", &BIND_ADDRESS)
	readEnvString("TMP_DIR", &TMP_DIR)
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
