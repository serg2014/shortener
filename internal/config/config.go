// Package config make cofiguration for app. Get gofig options from env and flags.
// Env has priority.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
)

// ServerAddress host and port
type ServerAddress struct {
	// Host is hostname where app will work
	Host string `json:"-"`
	// Port is the number of port where app will work
	Port uint64 `json:"-"`
}

// String flag.Value interface for type ServerAddress
func (sa *ServerAddress) String() string {
	return fmt.Sprintf("%s:%d", sa.Host, sa.Port)
}

// Set flag.Value interface for type ServerAddress
func (sa *ServerAddress) Set(flagValue string) error {
	host, portStr, err := net.SplitHostPort(flagValue)
	if err != nil {
		return err
	}

	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return err
	}
	sa.Host = host
	// TODO is it really need?
	if sa.Host == "" {
		sa.Host = "localhost"
	}
	sa.Port = port
	return nil
}

// UnmarshalJSON for ServerAddress
func (sa *ServerAddress) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	err = sa.Set(s)
	if err != nil {
		return err
	}
	return nil
}

type config struct {
	// ServerAddress is hostname where app will work
	ServerAddress ServerAddress `env:"SERVER_ADDRESS" json:"server_address"`
	// BaseURL - external hostname of the app
	BaseURL string `env:"BASE_URL" json:"base_url"`
	// LogLevel - logging level
	LogLevel string `env:"LOG_LEVEL" json:"log_level"`
	// FileStoragePath - path to the file where storage will save
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	// DatabaseDSN - dsn for connect ot database
	DatabaseDSN string `env:"DATABASE_DSN" json:"database_dsn"`
	// HTTPS use https
	HTTPS bool `env:"ENABLE_HTTPS" json:"enable_https"`
	// ConfigPath path to the config file json
	ConfigPath string `env:"CONFIG" json:"-"`
	// TrustedSubnet subnet for internal usage
	TrustedSubnet TrustedSubnet `env:"TRUSTED_SUBNET" json:"trusted_subnet"`
}

// newConfig create a new *config
func newConfig() *config {
	c := &config{}
	c.setDefaults()
	return c
}

func (c *config) setDefaults() {
	if c.ServerAddress.Host == "" {
		c.ServerAddress.Host = "localhost"
	}
	if c.ServerAddress.Port == 0 {
		c.ServerAddress.Port = 8080
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

// Config global var. use it as singleton
var Config = newConfig()

// URL - return BaseURL(if set) or default value.
func (c *config) URL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	proto := "http"
	if c.HTTPS {
		proto = "https"
	}
	return fmt.Sprintf("%s://%s:%d/", proto, c.ServerAddress.Host, c.ServerAddress.Port)
}

// InitConfig - initialize config
func (c *config) InitConfig() error {
	flag.Var(&c.ServerAddress, "a", "Net address host:port")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "Like http://ya.ru")
	flag.StringVar(&c.LogLevel, "l", c.LogLevel, "log level")
	flag.StringVar(&c.FileStoragePath, "f", c.FileStoragePath, "path to storage file")
	flag.StringVar(&c.DatabaseDSN, "d", c.DatabaseDSN, "database dsn")
	flag.BoolVar(&c.HTTPS, "s", c.HTTPS, "enable https")
	flag.StringVar(&c.ConfigPath, "c", "", "config path")
	flag.StringVar(&c.ConfigPath, "config", "", "config path")
	flag.Var(&c.TrustedSubnet, "t", "trusted subnet like (192.168.1.0/24)")
	flag.Parse()

	err := env.ParseWithOptions(
		c,
		env.Options{
			FuncMap: map[reflect.Type]env.ParserFunc{
				reflect.TypeOf(TrustedSubnet{}): func(val string) (any, error) {
					tsn := TrustedSubnet{}
					err := tsn.Set(val)
					if err != nil {
						return tsn, fmt.Errorf("%w: %w", ErrParseCIDR, err)
					}
					return tsn, nil
				},
				reflect.TypeOf(ServerAddress{}): func(val string) (any, error) {
					sa := ServerAddress{}
					err := sa.Set(val)
					if err != nil {
						return nil, err
					}
					return sa, nil
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("bad env: %w", err)
	}

	if c.BaseURL != "" {
		if !strings.HasSuffix(c.BaseURL, "/") {
			c.BaseURL = c.BaseURL + "/"
		}
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
		return nil, fmt.Errorf("can not open %s: %w", path, err)
	}

	var configFromFile config
	err = json.NewDecoder(f).Decode(&configFromFile)
	if err != nil {
		return nil, err
	}

	configFromFile.setDefaults()
	return &configFromFile, nil
}
