package main

import (
	"github.com/danushkaherath/klass-go/ginboot-cli/cmd"
	"log"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
