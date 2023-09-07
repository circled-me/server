package utils

import (
	"log"

	"github.com/gin-gonic/gin"
)

type errorLogWriter struct {
	gin.ResponseWriter
	gc *gin.Context
}

func (w errorLogWriter) Write(b []byte) (int, error) {
	status := w.gc.Writer.Status()
	if status >= 400 {
		log.Printf("[DEBUG ERROR]: Status %d, Body: %s", status, string(b))
	}
	return w.ResponseWriter.Write(b)
}

// ErrorLogMiddleware doesn't work with GZIP
func ErrorLogMiddleware(c *gin.Context) {
	blw := &errorLogWriter{gc: c, ResponseWriter: c.Writer}
	c.Writer = blw
	c.Next()
}
