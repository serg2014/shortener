// package config make cofiguration for app. Get gofig options from env and flags.
// Env has priority.
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
	// Host is hostname where app will work
	Host string `env:"SERVER_ADDRESS"`
	// Port is the number of port where app will work
	Port uint64
	// BaseURL - external hostname of the app
	BaseURL string `env:"BASE_URL"`
	// LogLevel - logging level
	LogLevel string `env:"LOG_LEVEL"`
	// FileStoragePath - path to the file where storage will save
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	// DatabaseDSN - dsn for connect ot database
	DatabaseDSN string `env:"DATABASE_DSN"`
}

// newConfig create a new *config
func newConfig() *config {
	return &config{
		Host:            "localhost",
		Port:            8080,
		BaseURL:         "",
		LogLevel:        "",
		FileStoragePath: "",
	}
}

// global var. use it as singleton
var Config = newConfig()

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

// URL - return BaseURL(if set) or default value.
func (c *config) URL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return fmt.Sprintf("http://%s:%d/", Config.Host, Config.Port)
}

// InitConfig - initialize config
func (c *config) InitConfig() error {
	flag.Var(c, "a", "Net address host:port")
	flag.StringVar(&c.BaseURL, "b", "", "Like http://ya.ru")
	flag.StringVar(&c.LogLevel, "l", "info", "log level")
	flag.StringVar(&c.FileStoragePath, "f", "", "path to storage file")
	flag.StringVar(&c.DatabaseDSN, "d", "", "database dsn")
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

	if envConfig.LogLevel != "" {
		c.LogLevel = envConfig.LogLevel
	}

	if envConfig.FileStoragePath != "" {
		c.FileStoragePath = envConfig.FileStoragePath
	}

	if envConfig.DatabaseDSN != "" {
		c.DatabaseDSN = envConfig.DatabaseDSN
	}

	return nil
}
