//go:build js && wasm

package main

import (
	"github.com/go-courier/rollup-plugin-go-wasm/examples/app/sub"
)

func main() {
	sub.Write("Hello Go Wasm 11")
}
