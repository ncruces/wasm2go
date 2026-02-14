package main

import "os"

func main() {
	translate(os.Args[1], os.Stdin, os.Stdout)
}
