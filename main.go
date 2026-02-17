package main

import (
	"log"
	"os"
)

func main() {
	err := translate(os.Args[1], os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
