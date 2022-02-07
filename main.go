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
	"strings"

	"github.com/gin-gonic/gin"
)

type ProxyEntry struct {
	Address     string `json:"address"`
	Path        string `json:"path"`
	MatchDomain string `json:"domain"`
}

type Config struct {
	Port               string       `json:"port"`
	HTTPPort           string       `json:"http_port"`
	Entries            []ProxyEntry `json:"entries"`
	NoRoute            string       `json:"noroute_route"`
	HTTPS              bool         `json:"https"`
	CertPath           string       `json:"cert_path"`
	PrivateKeyPath     string       `json:"private_key_path"`
	AlwaysHTTPS        bool         `json:"always_https"`
	DefaultHTTPSDomain string       `json:"default_https_domain"`
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
	entries := make(map[string]map[string]string)

	for _, entry := range config_obj.Entries {
		if _, ok := entries[entry.Path]; !ok {
			entries[entry.Path] = make(map[string]string)
		} else {
			entries[entry.Path][entry.MatchDomain] = entry.Address
		}
	}

	for k, p := range entries {
		addrMap := p
		path := k

		group := r.Group(path)

		group.Any("/*path", func(c *gin.Context) {
			s, found := addrMap[c.Request.Host]

			if !found {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			remote, err := url.Parse(s)

			if err != nil {
				panic(err)
			}

			proxy := httputil.NewSingleHostReverseProxy(remote)

			log.Println(c.Request.Host)
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

	for i := 0; i < len(config_obj.Entries); i++ {
		entry := config_obj.Entries[i]

		r.Any(fmt.Sprintf("%s/*path", entry.Path), func(c *gin.Context) {

			if strings.Compare(entry.MatchDomain, c.Request.Host) != 0 {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			remote, err := url.Parse(entry.Address)

			if err != nil {
				panic(err)
			}

			proxy := httputil.NewSingleHostReverseProxy(remote)

			log.Println(c.Request.Host)
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

	if len(config_obj.NoRoute) > 0 {
		r.NoRoute(func(c *gin.Context) {
			c.Redirect(http.StatusFound, config_obj.NoRoute)
		})
	}

	if config_obj.HTTPS {
		if config_obj.AlwaysHTTPS {
			go func() {
				h := gin.Default()
				h.GET("/*path", func(c *gin.Context) {
					path := c.Param("path")
					c.Redirect(http.StatusFound, "https://"+config_obj.DefaultHTTPSDomain+"/"+path)
				})
				h.Run(config_obj.HTTPPort)
			}()
		}

		r.RunTLS(config_obj.Port, config_obj.CertPath, config_obj.PrivateKeyPath)
	} else {
		r.Run(config_obj.Port)
	}
}
