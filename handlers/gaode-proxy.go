package handlers

import (
	"io"
	"net/http"
	"net/url"
	"server/config"
	"server/models"

	"log"

	"github.com/gin-gonic/gin"
)

func ProxyGaodeMapsTiles(c *gin.Context, user *models.User) {
	currentUrl, err := url.Parse(c.Request.RequestURI)
	if err != nil {
		log.Printf("Failed to parse request URI (%s): %v", c.Request.RequestURI, err)
		c.AbortWithStatus(400)
		return
	}
	queryParams := currentUrl.Query()
	subdomainParam := queryParams.Get("subdomain")
	queryParams.Del("subdomain")
	queryParams.Set("key", config.GAODE_API_KEY) // Set the server-side API key
	tileUrl := "https://webst0" + subdomainParam + ".is.autonavi.com/appmaptile?" + queryParams.Encode()
	// Create a new request to the tile URL
	resp, err := http.Get(tileUrl)
	if err != nil {
		log.Printf("Failed to fetch tile from Gaode Maps (%s): %v", tileUrl, err)
		c.AbortWithStatus(502)
		return
	}
	defer resp.Body.Close()

	// Copy the response headers
	for k, v := range resp.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)

	// Copy the response body
	_, _ = io.Copy(c.Writer, resp.Body)
}
