package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

type ProxyEntry struct {
	Address string `json:"address"`
	Path    string `json:"path"`
}

func main() {
	config_path := flag.String("config", "./config.json", "Config path")

	// Read the config file
	config, cErr := os.ReadFile(*config_path)

	if cErr != nil {
		log.Panicln(cErr)
	}

	var entries []ProxyEntry

	if jErr := json.Unmarshal(config, &entries); jErr != nil {
		log.Panicln(jErr)
	}

	if len(entries) == 0 {
		log.Panicf("No entries in %s", *config_path)
	}

	r := gin.Default()
	// Register handlers
	for i := 0; i < len(entries); i++ {
		entry := entries[i]

		r.Any(fmt.Sprintf("/*%s", entry.Path), func(c *gin.Context) {
			remote, err := url.Parse(entry.Address)

			if err != nil {
				panic(err)
			}

			proxy := httputil.NewSingleHostReverseProxy(remote)
			//Define the director func
			//This is a good place to log, for example
			proxy.Director = func(req *http.Request) {
				req.Header = c.Request.Header
				req.Host = remote.Host
				req.URL.Scheme = remote.Scheme
				req.URL.Host = remote.Host
				req.URL.Path = c.Param(entry.Path)[len(entry.Path):]
			}

			proxy.ServeHTTP(c.Writer, c.Request)
		})
	}

	/* r.GET("/", func(c *gin.Context) {
		c.String(200, "Hello")
	}) */

	r.Run(":8080")
}
