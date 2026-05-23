package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	output  = flag.String("o", "", "output file (default stdout)")
	pkgName = flag.String("pkg", "", "package name for the generated file (default module name, or wasm2go)")
	tags    = flag.String("tags", "", "go:build tags to include in the generated file")

	embed  = flag.Bool("embed", false, "go:embed data sections from a .dat file")
	nanbox = flag.Bool("nanbox", false, "attempt to canonicalize NaNs")
	nohost = flag.Bool("nohost", false, "don't generate interfaces for imports")
	noopt  = flag.Bool("noopt", false, "disable all optimization passes")
	unsafe = flag.Bool("unsafe", false, "allow importing unsafe")

	provided  stringFlags
	embedFile string
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("wasm2go: ")

	flag.Var(&provided, "provided", "file containing provided import functions")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [option]... [input.wasm]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}

	if *embed {
		if *output == "" {
			log.Fatal("-embed requires `-o output.go` to be specified")
		}
		embedFile = strings.TrimSuffix(*output, filepath.Ext(*output)) + ".dat"
	}

	in := os.Stdin
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		in = f
	}

	out := os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		out = f
	}

	if err := translate(in, out); err != nil {
		log.Fatal(err)
	}
	if err := out.Close(); err != nil {
		log.Fatal(err)
	}
}

type stringFlags []string

func (l *stringFlags) String() string {
	return strings.Join(*l, ", ")
}

func (l *stringFlags) Set(value string) error {
	*l = append(*l, value)
	return nil
}

var seenReturnCall bool

func warnReturnCall() {
	if !seenReturnCall {
		seenReturnCall = true
		log.Print("return_call does not guarantee tail behavior")
	}
}

func needsUnsafe(msg string) {
	if !*unsafe {
		log.Fatal("needs unsafe: " + msg)
	}
}
