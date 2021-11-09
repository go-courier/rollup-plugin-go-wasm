//go:build js && wasm

package main

import "syscall/js"

func main() {
	js.Global().Get("document").Call("write", "Hello Go Wasm 11")
}
