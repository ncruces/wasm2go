package main

import (
	"flag"
	"log"
	"os"
)

var (
	embed  = flag.String("embed", "", "go:embed data sections from this file")
	nanbox = flag.Bool("nanbox", false, "whether to try to canonicalize NaNs")
	nohost = flag.Bool("nohost", false, "disable generating interfaces for imports")
	noopt  = flag.Bool("noopt", false, "disable all optimization passes")
	unsafe = flag.Bool("unsafe", false, "allow importing unsafe")
)

func main() {
	flag.Parse()
	err := translate(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
