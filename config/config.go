package config

import (
	"os"
	"strconv"
	"strings"
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
	GAODE_API_KEY              = ""     // Gaode Maps API key, optional
	DEBUG_MODE                 = true
	FACE_DETECT                = true  // Enable/disable face detection
	FACE_DETECT_CNN            = false // Use Convolutional Neural Network for face detection (as opposed to HOG). Much slower, supposedly more accurate at different angles
	FACE_MAX_DISTANCE_SQ       = 0.11  // Squared distance between faces to consider them similar
	// TURN server support is better be enabled if you are planning to use the video/audio call functionalities.
	// By default a public STUN server would be added, but in cases where NAT firewall rules are too strict (symmetric NATs, etc), a TURN server is needed to relay the traffic
	TURN_SERVER_IP        = ""   // If configured, Pion TURN server would be started locally and this value used to advertise ourselves. Should be your public IP. Defaults to empty string.
	TURN_SERVER_PORT      = 3478 // Defaults to UDP port 3478
	TURN_TRAFFIC_MIN_PORT = 49152
	TURN_TRAFFIC_MAX_PORT = 65535 // Advertise-able UDP port range for TURN traffic. Those ports need to be open on your public IP (and forwarded to the circled.me server instance). Defaults to 49152-65535
)

func init() {
	readEnvString("TLS_DOMAINS", &TLS_DOMAINS)
	readEnvString("PUSH_SERVER", &PUSH_SERVER)
	readEnvString("MYSQL_DSN", &MYSQL_DSN)
	readEnvString("SQLITE_FILE", &SQLITE_FILE)
	readEnvString("BIND_ADDRESS", &BIND_ADDRESS)
	readEnvString("TMP_DIR", &TMP_DIR)
	readEnvString("DEFAULT_BUCKET_DIR", &DEFAULT_BUCKET_DIR)
	readEnvString("GAODE_API_KEY", &GAODE_API_KEY)
	readEnvString("DEFAULT_ASSET_PATH_PATTERN", &DEFAULT_ASSET_PATH_PATTERN)
	readEnvBool("DEBUG_MODE", &DEBUG_MODE)
	readEnvBool("FACE_DETECT", &FACE_DETECT)
	readEnvBool("FACE_DETECT_CNN", &FACE_DETECT_CNN)
	readEnvFloat("FACE_MAX_DISTANCE_SQ", &FACE_MAX_DISTANCE_SQ)
	readEnvString("TURN_SERVER_IP", &TURN_SERVER_IP)
	readEnvInt("TURN_SERVER_PORT", &TURN_SERVER_PORT)
	readEnvInt("TURN_TRAFFIC_MIN_PORT", &TURN_TRAFFIC_MIN_PORT)
	readEnvInt("TURN_TRAFFIC_MAX_PORT", &TURN_TRAFFIC_MAX_PORT)
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

func readEnvInt(name string, value *int) {
	v := os.Getenv(name)
	if v == "" {
		return
	}
	f, err := strconv.Atoi(v)
	if err != nil {
		return
	}
	*value = f
}
