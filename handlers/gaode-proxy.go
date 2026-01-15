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
	// Validate the subdomain parameter to prevent SSRF by restricting it to expected values.
	if subdomainParam == "" {
		// Use a safe default if no subdomain is provided.
		subdomainParam = "1"
	} else {
		switch subdomainParam {
		case "1", "2", "3", "4":
			// allowed
		default:
			log.Printf("Invalid subdomain parameter (%s) in request URI (%s)", subdomainParam, c.Request.RequestURI)
			c.AbortWithStatus(400)
			return
		}
	}
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
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		log.Printf("Failed to copy response body from Gaode Maps (%s): %v", tileUrl, err)
	}
}
