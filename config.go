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
		cortezaAuth  string
		jwtSecret    []byte
		clientKey    string
		clientSecret string
	}
)

const (
	envKeyHttpAddr     = "HTTP_ADDR"
	envKeyEsAddr       = "ES_ADDRESS"
	envKeyBaseUrl      = "CORTEZA_SERVER_BASE_URL"
	envKeyAuthUrl      = "CORTEZA_SERVER_AUTH_URL"
	envKeyJwtSecret    = "CORTEZA_SERVER_JWT_SECRET"
	envKeyClientKey    = "CORTEZA_SERVER_CLIENT_KEY"
	envKeyClientSecret = "CORTEZA_SERVER_CLIENT_SECRET"
)

func getConfig() (*config, error) {
	c := &config{}

	return c, func() error {

		baseUrl := options.EnvString(envKeyBaseUrl, "http://server:80")

		c.httpAddr = options.EnvString(envKeyHttpAddr, "127.0.0.1:3101")

		c.cortezaAuth = options.EnvString(envKeyAuthUrl, baseUrl+"/auth")
		if c.cortezaAuth == "" {
			return fmt.Errorf("endpount URL for corteza auth  (%s) is empty or missing", envKeyAuthUrl)
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

		for _, a := range strings.Split(options.EnvString("ES_ADDRESS", envKeyEsAddr), " ") {
			if a = strings.TrimSpace(a); a != "" {
				c.es.addresses = append(c.es.addresses, a)
			}
		}

		return nil
	}()
}
