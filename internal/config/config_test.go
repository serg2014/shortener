package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Очистка флагов
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

// Очистка переменных окружения
func unsetEnvVars() {
	os.Unsetenv("SERVER_ADDRESS")
	os.Unsetenv("BASE_URL")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("FILE_STORAGE_PATH")
	os.Unsetenv("DATABASE_DSN")
	os.Unsetenv("ENABLE_HTTPS")
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name    string
		expect  *config
		envVars map[string]string
		args    []string
	}{
		{
			name: "no env. no flag",
			expect: &config{
				Host:            "localhost",
				Port:            8080,
				BaseURL:         "",
				LogLevel:        "info",
				FileStoragePath: "",
				DatabaseDSN:     "",
			},
			envVars: make(map[string]string),
			args:    make([]string, 0),
		},
		{
			name: "only env",
			expect: &config{
				Host:            "localhost2",
				Port:            9090,
				BaseURL:         "http://some.host/",
				LogLevel:        "debug",
				FileStoragePath: "/file/path",
				DatabaseDSN:     "dsn",
				HTTPS:           true,
			},
			envVars: map[string]string{
				"SERVER_ADDRESS":    "localhost2:9090",
				"BASE_URL":          "http://some.host",
				"LOG_LEVEL":         "debug",
				"FILE_STORAGE_PATH": "/file/path",
				"DATABASE_DSN":      "dsn",
				"ENABLE_HTTPS":      "true",
			},
			args: make([]string, 0),
		},
		{
			name: "only flags",
			expect: &config{
				Host:            "localhost2",
				Port:            9090,
				BaseURL:         "http://some.host/",
				LogLevel:        "debug",
				FileStoragePath: "/file/path",
				DatabaseDSN:     "dsn",
				HTTPS:           true,
			},
			envVars: make(map[string]string),
			args: []string{
				"-a=localhost2:9090",
				"-b=http://some.host",
				"-l=debug",
				"-f=/file/path",
				"-d=dsn",
				"-s=1",
			},
		},
		{
			name: "env and flags. env priority",
			expect: &config{
				Host:            "localhost3",
				Port:            1010,
				BaseURL:         "http://host.some/",
				LogLevel:        "error",
				FileStoragePath: "/path/file",
				DatabaseDSN:     "dsn-env",
				HTTPS:           true,
			},
			envVars: map[string]string{
				"SERVER_ADDRESS":    "localhost3:1010",
				"BASE_URL":          "http://host.some",
				"LOG_LEVEL":         "error",
				"FILE_STORAGE_PATH": "/path/file",
				"DATABASE_DSN":      "dsn-env",
				"ENABLE_HTTPS":      "true",
			},
			args: []string{
				"-a=localhost2:9090",
				"-b=http://some.host",
				"-l=debug",
				"-f=/file/path",
				"-d=dsn",
				"-s=0",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Очистка состояния
			resetFlags()
			unsetEnvVars()

			if len(test.envVars) != 0 {
				for k := range test.envVars {
					t.Setenv(k, test.envVars[k])
				}
			}
			// Установка аргументов командной строки
			os.Args = append([]string{"cmd"}, test.args...)

			conf := newConfig()
			err := conf.InitConfig()
			require.NoError(t, err)
			assert.Equal(t, test.expect, conf)
		})
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name    string
		baseurl string
		expect  string
	}{
		{
			name:    "with base url",
			baseurl: "http://some.host",
			expect:  "http://some.host/",
		},
		{
			name:    "with base url with slash",
			baseurl: "http://some.host2/",
			expect:  "http://some.host2/",
		},
		{
			name:    "without base url",
			baseurl: "",
			expect:  "http://localhost:8080/",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetFlags()
			unsetEnvVars()
			if len(test.baseurl) != 0 {
				t.Setenv("BASE_URL", test.baseurl)
			}
			os.Args = []string{"cmd"}

			conf := newConfig()
			err := conf.InitConfig()
			require.NoError(t, err)
			assert.Equal(t, test.expect, conf.URL())
		})
	}
}
