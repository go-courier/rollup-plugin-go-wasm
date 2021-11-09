package main

import (
	"flag"
	"reflect"
	"strconv"
)

type Flags struct {
	ImportWasmSuffix string `name:"import-wasm-suffix" summary:"import index.wasm will with this suffix in the wrapper called index.mjs"`
	Watch            bool   `name:"watch" summary:"watch mode, will auto rebuild when files changes in this module"`
	List             bool   `name:"list" summary:"output dep files"`
}

func (f *Flags) BindTo(fs *flag.FlagSet) {
	rv := reflect.ValueOf(f).Elem()
	tpe := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ft := tpe.Field(i)

		name := ft.Tag.Get("name")
		defaultValue := ft.Tag.Get("default")
		summary := ft.Tag.Get("summary")

		switch fv.Kind() {
		case reflect.Bool:
			d, _ := strconv.ParseBool(defaultValue)
			fs.BoolVar(fv.Addr().Interface().(*bool), name, d, summary)
		case reflect.String:
			fs.StringVar(fv.Addr().Interface().(*string), name, defaultValue, summary)
		}
	}
}
