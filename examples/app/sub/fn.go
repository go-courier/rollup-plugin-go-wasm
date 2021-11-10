//go:build js && wasm

package sub

import "syscall/js"

func Write(s string) {
	js.Global().Get("document").Call("write", s)
}
