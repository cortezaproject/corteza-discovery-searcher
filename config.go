package main

import (
	"fmt"
	"github.com/cortezaproject/corteza-server/pkg/options"
	_ "github.com/joho/godotenv/autoload"
	"os"
	"strings"
)

type (
	config struct {
		httpAddr string
		es       struct {
			addresses []string
		}
		cortezaHttp  string
		cortezaAuth  string
		jwtSecret    []byte
		clientKey    string
		clientSecret string
	}
)

const (
	discoverySearcher  = "DISCOVERY_SEARCHER_"
	envKeyHttpAddr     = discoverySearcher + "HTTP_ADDR"
	envKeyEsAddr       = discoverySearcher + "ES_ADDRESS"
	envKeyBaseUrl      = discoverySearcher + "CORTEZA_SERVER_BASE_URL"
	envKeyAuthUrl      = discoverySearcher + "CORTEZA_SERVER_AUTH_URL"
	envKeyJwtSecret    = discoverySearcher + "CORTEZA_SERVER_JWT_SECRET"
	envKeyClientKey    = discoverySearcher + "CORTEZA_SERVER_CLIENT_KEY"
	envKeyClientSecret = discoverySearcher + "CORTEZA_SERVER_CLIENT_SECRET"
)

func getConfig() (*config, error) {
	c := &config{}

	return c, func() error {

		c.cortezaHttp = options.EnvString(envKeyBaseUrl, "http://server:80")
		if c.cortezaHttp == "" {
			return fmt.Errorf("endpoint URL for corteza (%s) is empty or missing", envKeyAuthUrl)
		}

		c.httpAddr = options.EnvString(envKeyHttpAddr, "127.0.0.1:3101")

		c.cortezaAuth = options.EnvString(envKeyAuthUrl, c.cortezaHttp+"/auth")
		if c.cortezaAuth == "" {
			return fmt.Errorf("endpoint URL for corteza auth (%s) is empty or missing", envKeyAuthUrl)
		}

		if tmp := os.Getenv(envKeyJwtSecret); tmp != "" {
			c.jwtSecret = []byte(tmp)
		}

		if c.clientKey = os.Getenv(envKeyClientKey); c.clientKey == "" {
			return fmt.Errorf("client key (%s) is empty or missing", envKeyClientKey)
		}

		if c.clientSecret = os.Getenv(envKeyClientSecret); c.clientSecret == "" {
			return fmt.Errorf("client secret (%s) is empty or missing", envKeyClientSecret)
		}

		for _, a := range strings.Split(options.EnvString(envKeyEsAddr, "http://localhost:9200"), " ") {
			if a = strings.TrimSpace(a); a != "" {
				c.es.addresses = append(c.es.addresses, a)
			}
		}

		return nil
	}()
}
