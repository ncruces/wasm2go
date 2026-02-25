package main

import (
	"log"
	"os"
)

func main() {
	err := translate(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
