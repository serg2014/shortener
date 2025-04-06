package app

import "fmt"

const (
	Port = 8080
	Host = "localhost"
)

func Url() string {
	return fmt.Sprintf("http://%s:%d/", Host, Port)
}
