package main

import (
	"github.com/klass-lk/ginboot-cli/cmd"
	"log"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
