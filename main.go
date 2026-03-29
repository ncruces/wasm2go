package main

import (
	"flag"
	"log"
	"os"
)

var (
	nanbox = flag.Bool("nanbox", false, "whether to try to canonicalize NaNs")
	nohead = flag.Bool("nohead", false, "disable the header comment (including build tags)")
	nohost = flag.Bool("nohost", false, "disable generating interfaces for imports")
	noopt  = flag.Bool("noopt", false, "disable all optimization passes")
)

func main() {
	flag.Parse()
	err := translate(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
