package config

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v6"
)

type config struct {
	Host    string `env:"SERVER_ADDRESS"`
	Port    uint64
	BaseURL string `env:"BASE_URL"`
}

var Config = &config{
	Host:    "localhost",
	Port:    8080,
	BaseURL: "",
}

func (c *config) String() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
func (c *config) Set(flagValue string) error {
	host, portStr, err := net.SplitHostPort(flagValue)
	if err != nil {
		return err
	}

	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return err
	}
	c.Host = host
	if c.Host == "" {
		c.Host = "localhost"
	}
	c.Port = port
	return nil
}

func (c *config) URL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return fmt.Sprintf("http://%s:%d/", Config.Host, Config.Port)
}

func (c *config) InitConfig() error {
	flag.Var(c, "a", "Net address host:port")
	flag.StringVar(&c.BaseURL, "b", "", "Like http://ya.ru")
	flag.Parse()

	var envConfig config
	err := env.Parse(&envConfig)
	if err != nil {
		return err
	}
	if envConfig.Host != "" {
		err := c.Set(envConfig.Host)
		if err != nil {
			return err
		}
	}

	if envConfig.BaseURL != "" {
		c.BaseURL = envConfig.BaseURL
	}

	if c.BaseURL != "" {
		if !strings.HasSuffix(c.BaseURL, "/") {
			c.BaseURL = c.BaseURL + "/"
		}
	}

	return nil
}
