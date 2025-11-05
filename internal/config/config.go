// Package config make cofiguration for app. Get gofig options from env and flags.
// Env has priority.
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v6"
)

var (
	configTrue    = true
	configTruePtr = &configTrue

	configFalse    = false
	configFalsePtr = &configFalse
)

type config struct {
	// Host is hostname where app will work
	Host string `env:"SERVER_ADDRESS" json:"server_address"`
	// BaseURL - external hostname of the app
	BaseURL string `env:"BASE_URL" json:"base_url"`
	// LogLevel - logging level
	LogLevel string `env:"LOG_LEVEL" json:"log_level"`
	// FileStoragePath - path to the file where storage will save
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	// DatabaseDSN - dsn for connect ot database
	DatabaseDSN string `env:"DATABASE_DSN" json:"database_dsn"`
	// Port is the number of port where app will work
	Port uint64 `json:"-"`
	// HTTPS use https
	HTTPS      *bool  `env:"ENABLE_HTTPS" json:"enable_https"`
	ConfigPath string `env:"CONFIG" json:"-"`
}

// newConfig create a new *config
func newConfig() *config {
	c := &config{}
	c.setDefaults()
	return c
}

func (c *config) setDefaults() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

// Config global var. use it as singleton
var Config = newConfig()

// String flag.Value interface
func (c *config) String() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Set flag.Value interface
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
	proto := "http"
	if c.HTTPS != nil && *c.HTTPS {
		proto = "https"
	}
	return fmt.Sprintf("%s://%s:%d/", proto, c.Host, c.Port)
}

// InitConfig - initialize config
func (c *config) InitConfig() error {
	flag.Var(c, "a", "Net address host:port")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "Like http://ya.ru")
	flag.StringVar(&c.LogLevel, "l", c.LogLevel, "log level")
	flag.StringVar(&c.FileStoragePath, "f", c.FileStoragePath, "path to storage file")
	flag.StringVar(&c.DatabaseDSN, "d", c.DatabaseDSN, "database dsn")
	flag.BoolFunc("s", "enable https", func(val string) error {
		v := strings.ToLower(val)
		switch v {
		case "1", "true", "t":
			c.HTTPS = configTruePtr
		case "0", "false", "f":
			c.HTTPS = configFalsePtr
		default:
			return errors.New("can not parse bool")
		}
		return nil
	})
	flag.StringVar(&c.ConfigPath, "c", "", "config path")
	flag.StringVar(&c.ConfigPath, "config", "", "config path")
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

	if envConfig.HTTPS != nil {
		c.HTTPS = envConfig.HTTPS
	}

	if envConfig.ConfigPath != "" {
		c.ConfigPath = envConfig.ConfigPath
	}

	if c.ConfigPath != "" {
		newconfig, err := configFromFileWithFlags(c)
		if err != nil {
			return err
		}
		*c = *newconfig
	}
	return nil
}

func configFromFileWithFlags(c *config) (*config, error) {
	newconfig, err := getConfigFromFile(c.ConfigPath)
	if err != nil {
		return nil, err
	}
	newconfig.ConfigPath = c.ConfigPath
	// сбросить переменную окружения CONFIG и флаги -c -config
	os.Unsetenv("CONFIG")
	newArgs := make([]string, 0, len(os.Args))
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "-c" || os.Args[i] == "-config" {
			i++
			continue
		}
		newArgs = append(newArgs, os.Args[i])
	}
	os.Args = newArgs
	// reset flag
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	err = newconfig.InitConfig()
	if err != nil {
		return nil, err
	}
	newconfig.ConfigPath = c.ConfigPath
	return newconfig, nil
}

func getConfigFromFile(path string) (*config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var configFromFile config
	dec := json.NewDecoder(f)
	err = dec.Decode(&configFromFile)
	if err != nil {
		return nil, err
	}
	if configFromFile.Host != "" {
		err = configFromFile.Set(configFromFile.Host)
		if err != nil {
			return nil, err
		}
	}
	configFromFile.setDefaults()
	return &configFromFile, nil
}
