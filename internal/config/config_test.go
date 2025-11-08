package config

import (
	"flag"
	"net"
	"os"
	"path"
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
	tmpDir, err := os.MkdirTemp("", "config")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		expect     *config
		envVars    map[string]string
		args       []string
		configData string
		configPath string
		err        error
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
				HTTPS:           nil,
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
				HTTPS:           configTruePtr,
				TrustedSubnet: &TrustedSubnet{
					IP:   net.IP([]byte{0xc0, 0xa8, 0x01, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			envVars: map[string]string{
				"SERVER_ADDRESS":    "localhost2:9090",
				"BASE_URL":          "http://some.host",
				"LOG_LEVEL":         "debug",
				"FILE_STORAGE_PATH": "/file/path",
				"DATABASE_DSN":      "dsn",
				"ENABLE_HTTPS":      "true",
				"TRUSTED_SUBNET":    "192.168.1.0/24",
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
				HTTPS:           configTruePtr,
				TrustedSubnet: &TrustedSubnet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			envVars: make(map[string]string),
			args: []string{
				"-a=localhost2:9090",
				"-b=http://some.host",
				"-l=debug",
				"-f=/file/path",
				"-d=dsn",
				"-s",
				"-t=127.0.0.1/24",
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
				HTTPS:           configFalsePtr,
				TrustedSubnet: &TrustedSubnet{
					IP:   net.IP([]byte{0xc0, 0xa8, 0x01, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			envVars: map[string]string{
				"SERVER_ADDRESS":    "localhost3:1010",
				"BASE_URL":          "http://host.some",
				"LOG_LEVEL":         "error",
				"FILE_STORAGE_PATH": "/path/file",
				"DATABASE_DSN":      "dsn-env",
				"ENABLE_HTTPS":      "false",
				"TRUSTED_SUBNET":    "192.168.1.0/24",
			},
			args: []string{
				"-a=localhost2:9090",
				"-b=http://some.host",
				"-l=debug",
				"-f=/file/path",
				"-d=dsn",
				"-s",
				"-t=127.0.0.1/24",
			},
		},
		{
			name: "no env. no flag. flag config",
			expect: &config{
				Host:            "localhost2",
				Port:            8081,
				BaseURL:         "",
				LogLevel:        "info",
				FileStoragePath: "",
				DatabaseDSN:     "",
				HTTPS:           nil,
				ConfigPath:      path.Join(tmpDir, "config1.json"),
				TrustedSubnet: &TrustedSubnet{
					IP:   net.IP([]byte{0xc0, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			envVars:    make(map[string]string),
			args:       make([]string, 0),
			configPath: path.Join(tmpDir, "config1.json"),
			configData: `{"server_address":"localhost2:8081", "trusted_subnet":"192.0.0.0/24"}`,
		},
		{
			name: "no env. no flag. flag config 2",
			expect: &config{
				Host:            "localhost",
				Port:            8080,
				BaseURL:         "",
				LogLevel:        "debug",
				FileStoragePath: "",
				DatabaseDSN:     "",
				HTTPS:           nil,
				ConfigPath:      path.Join(tmpDir, "config2.json"),
			},
			envVars:    make(map[string]string),
			args:       make([]string, 0),
			configPath: path.Join(tmpDir, "config2.json"),
			configData: `{"log_level":"debug"}`,
		},
		{
			name: "env. flag. flag config 3",
			expect: &config{
				Host:            "localhost3",
				Port:            1010,
				BaseURL:         "",
				LogLevel:        "debug",
				FileStoragePath: "",
				DatabaseDSN:     "",
				HTTPS:           configFalsePtr,
				ConfigPath:      path.Join(tmpDir, "config3.json"),
				TrustedSubnet: &TrustedSubnet{
					IP:   net.IP([]byte{0x7f, 0x0, 0x0, 0x0}),
					Mask: net.IPMask([]byte{0xff, 0xff, 0xff, 0x0}),
				},
			},
			envVars: map[string]string{
				"SERVER_ADDRESS": "localhost3:1010",
			},
			args: []string{
				"-a=localhost1:9090",
				"-s=0",
				"-t=127.0.0.1/24",
			},
			configPath: path.Join(tmpDir, "config3.json"),
			configData: `{"log_level":"debug","server_address":"localhost2:8081","enable_https":true,"trusted_subnet":"192.0.0.0/24"}`,
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
			if len(test.configData) != 0 {
				err = os.WriteFile(test.configPath, []byte(test.configData), 0600)
				require.NoError(t, err)
				test.args = append(test.args, "-config", test.configPath)
			}
			// Установка аргументов командной строки
			os.Args = append([]string{"cmd"}, test.args...)

			conf := newConfig()
			err := conf.InitConfig()
			if test.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, test.err)
			}
			assert.Equal(t, test.expect, conf)
		})
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		expect  string
	}{
		{
			name: "with base url",
			envVars: map[string]string{
				"BASE_URL": "http://some.host",
			},
			expect: "http://some.host/",
		},
		{
			name: "with base url with slash",
			envVars: map[string]string{
				"BASE_URL": "http://some.host2/",
			},
			expect: "http://some.host2/",
		},
		{
			name:    "without base url",
			envVars: map[string]string{},
			expect:  "http://localhost:8080/",
		},
		{
			name: "with server_address",
			envVars: map[string]string{
				"SERVER_ADDRESS": "localhost3:1010",
			},
			expect: "http://localhost3:1010/",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetFlags()
			unsetEnvVars()
			if len(test.envVars) != 0 {
				for k := range test.envVars {
					t.Setenv(k, test.envVars[k])
				}
			}
			os.Args = []string{"cmd"}

			conf := newConfig()
			err := conf.InitConfig()
			require.NoError(t, err)
			assert.Equal(t, test.expect, conf.URL())
		})
	}
}

func Test_getConfigFromFile(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		expect config
	}{
		{
			name: "do not use Port and ConfigPath from json",
			data: `{"server_address":"localhost2:8081","base_url": "https://localhost", "log_level": "debug", "Port": 90, "enable_https": true, "ConfigPath":"test"}`,
			expect: config{
				Host:       "localhost2",
				Port:       8081,
				BaseURL:    "https://localhost",
				LogLevel:   "debug",
				HTTPS:      configTruePtr,
				ConfigPath: "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "config")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			path := path.Join(tmpDir, "config.json")
			err = os.WriteFile(path, []byte(test.data), 0600)
			require.NoError(t, err)
			conf, err := getConfigFromFile(path)
			require.NoError(t, err)
			assert.Equal(t, test.expect, *conf)
		})
	}
}
