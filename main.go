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

type Config struct {
	Port           string       `json:"port"`
	Entries        []ProxyEntry `json:"entries"`
	HTTPS          bool         `json:"https"`
	CertPath       string       `json:"cert_path"`
	PrivateKeyPath string       `json:"private_key_path"`
}

func main() {
	config_path := flag.String("config", "./config.json", "Config path")

	// Read the config file
	config, cErr := os.ReadFile(*config_path)

	if cErr != nil {
		log.Panicln(cErr)
	}

	var config_obj Config

	if jErr := json.Unmarshal(config, &config_obj); jErr != nil {
		log.Panicln(jErr)
	}

	if len(config_obj.Entries) == 0 {
		log.Panicf("No entries in %s", *config_path)
	}

	r := gin.Default()
	// Register handlers
	for i := 0; i < len(config_obj.Entries); i++ {
		entry := config_obj.Entries[i]

		r.Any(fmt.Sprintf("%s/*path", entry.Path), func(c *gin.Context) {
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
				req.URL.Path = c.Param("path")
			}

			proxy.ServeHTTP(c.Writer, c.Request)
		})
	}

	r.GET("/", func(c *gin.Context) {
		c.String(200, "Hello")
	})

	if config_obj.HTTPS {
		r.RunTLS(config_obj.Port, config_obj.CertPath, config_obj.PrivateKeyPath)
	} else {
		r.Run(config_obj.Port)
	}
}
