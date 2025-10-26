package main

import "os"

func main() {
	os.Exit(0)       // want "os.Exit in func main, package main"
	defer os.Exit(0) // want "os.Exit in func main, package main"
	go os.Exit(0)    // want "os.Exit in func main, package main"
}

func one() {
	os.Exit(0)
	defer os.Exit(0)
	go os.Exit(0)
}
