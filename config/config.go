package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	FaceThresholdSquared = 0.36 // 0.6^2
)

var (
	TLS_DOMAINS                = ""                    // e.g. "example.com,example2.com"
	DEFAULT_ASSET_PATH_PATTERN = "<year>/<month>/<id>" // also available: <name>, <Month>
	PUSH_SERVER                = "https://push.circled.me"
	MYSQL_DSN                  = "" // MySQL will be used if this is set
	SQLITE_FILE                = "" // SQLite will be used if MYSQL_DSN is not configured and this is set
	BIND_ADDRESS               = "0.0.0.0:8080"
	TMP_DIR                    = "/tmp" // Used for temporary video conversion, etc (in case of S3 bucket)
	DEFAULT_BUCKET_DIR         = ""     // Used for creating initial bucket
	DEBUG_MODE                 = true
	FACE_DETECT_CNN            = true // Use Convolutional Neural Network for face detection (as opposed to HOG). Much slower, supposedly more accurate at different angles
	FACE_MAX_DISTANCE_SQ       = 0.11 // Squared distance between faces to consider them similar
)

func init() {
	readEnvString("TLS_DOMAINS", &TLS_DOMAINS)
	readEnvString("PUSH_SERVER", &PUSH_SERVER)
	readEnvString("MYSQL_DSN", &MYSQL_DSN)
	readEnvString("SQLITE_FILE", &SQLITE_FILE)
	readEnvString("BIND_ADDRESS", &BIND_ADDRESS)
	readEnvString("TMP_DIR", &TMP_DIR)
	readEnvString("DEFAULT_BUCKET_DIR", &DEFAULT_BUCKET_DIR)
	readEnvString("DEFAULT_ASSET_PATH_PATTERN", &DEFAULT_ASSET_PATH_PATTERN)
	readEnvBool("DEBUG_MODE", &DEBUG_MODE)
	readEnvBool("FACE_DETECT_CNN", &FACE_DETECT_CNN)
	readEnvFloat("FACE_MAX_DISTANCE_SQ", &FACE_MAX_DISTANCE_SQ)
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

func readEnvFloat(name string, value *float64) {
	v := os.Getenv(name)
	if v == "" {
		return
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return
	}
	*value = f
}
