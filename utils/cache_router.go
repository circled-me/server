package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	CacheNoCache = 0
	CacheCustom  = -1
)

type CacheRouter struct {
	CacheTime int // defaults to CacheNoCache = 0
}

func (cr *CacheRouter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cr.CacheTime != CacheCustom {
			if cr.CacheTime == CacheNoCache {
				c.Header("cache-control", "no-cache")
			} else {
				c.Header("cache-control", "private, max-age="+strconv.Itoa(cr.CacheTime))
			}
		}
		c.Next()
	}
}
