package main

import (
	"flag"
)

func main() {
	flag.Parse()
	hub := newHub()
	go hub.run()
}