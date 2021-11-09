package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

func main() {
	flags := Flags{}
	flags.BindTo(flag.CommandLine)

	flag.Parse()
	args := flag.Args()

	l := stdr.New(log.New(os.Stdout, "[go-wasm-pack] ", log.Lmsgprefix))

	if len(args) > 0 {
		ctx := logr.NewContext(context.Background(), l)
		b := NewWASMBuilder(args[0], flags)
		b.Start(ctx)
	}
}
