package app

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type config struct {
	Host string
	Port uint64
}

var NewConfig = &config{
	Host: "localhost",
	Port: 8080,
}

var NewURL = ""

func (c *config) String() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
func (c *config) Set(flagValue string) error {
	hp := strings.Split(flagValue, ":")
	if len(hp) != 2 {
		return errors.New("need address in a form host:port")
	}
	port, err := strconv.ParseUint(hp[1], 10, 32)
	if err != nil {
		return err
	}
	c.Host = hp[0]
	c.Port = port
	return nil
}

func URL() string {
	if NewURL != "" {
		return NewURL
	}
	return fmt.Sprintf("http://%s:%d/", NewConfig.Host, NewConfig.Port)
}
