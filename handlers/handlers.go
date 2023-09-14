package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Response struct {
	Error string `json:"error"`
}

type MultiResponse struct {
	Error  string   `json:"error"`
	Failed []uint64 `json:"failed"`
}

const (
	etagHeader = "ETag"
)

var (
	// Predefined errors
	OKResponse       = Response{}
	NopeResponse     = Response{"nope"}
	Nope2Response    = Response{"no no"}
	Nope3Response    = Response{"no no no"}
	DBError1Response = Response{"DB Error 1"}
	DBError2Response = Response{"DB Error 2"}
	DBError3Response = Response{"DB Error 3"}
	DBError4Response = Response{"DB Error 4"}
	OKMultiResponse  = MultiResponse{"", []uint64{}}
)

func isNotModified(c *gin.Context, tx *gorm.DB) bool {
	// Set the current ETag in all cases
	row := tx.Row()
	lastUpdatedAt := uint64(0)
	if row.Scan(&lastUpdatedAt) != nil {
		return false
	}
	c.Header("cache-control", "private, max-age=1")
	c.Header(etagHeader, strconv.FormatUint(lastUpdatedAt, 10))

	// ETag contains last updated asset time
	remoteLastUpdatedAt, _ := strconv.ParseUint(c.Request.Header.Get("If-None-Match"), 10, 64)
	if remoteLastUpdatedAt == lastUpdatedAt {
		c.Status(http.StatusNotModified)
		return true
	}
	return false
}
