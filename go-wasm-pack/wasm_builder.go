package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"golang.org/x/tools/go/packages"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func NewWASMBuilder(input string, flags Flags) *WASMBuilder {
	return &WASMBuilder{input: input, Flags: flags}
}

type WASMBuilder struct {
	Flags
	w            *fsnotify.Watcher
	input        string
	watchedPaths map[string]bool
}

func (b *WASMBuilder) Start(ctx context.Context) {
	if b.Flags.Watch {
		b.Watch(ctx)
		return
	}
	b.Build(ctx)
	return
}

func (b *WASMBuilder) Watch(ctx context.Context) {
	log := logr.FromContextOrDiscard(ctx)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err, "new fs watcher failed")
		return
	}
	b.w = watcher

	defer func() {
		_ = b.w.Close()
	}()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					b.Build(ctx)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error(err, "watch error")
			}
		}
	}()

	b.Build(ctx)

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh
}

func (b *WASMBuilder) Build(ctx context.Context) {
	log := logr.FromContextOrDiscard(ctx)

	pkg, paths, err := b.load()
	if err != nil {
		log.Error(err, "load failed")
		return
	}

	if b.List {
		_ = json.NewEncoder(os.Stdout).Encode(paths)
		return
	}

	if b.w != nil {
		b.registerPathsToWatch(paths)
	}

	if err := b.pkgWASM(ctx, pkgDir(pkg.Module, pkg.PkgPath)); err != nil {
		log.Error(err, "build wasm fail")
	}
}

func (b *WASMBuilder) load() (*packages.Package, map[string]bool, error) {
	loadedPackages, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedModule,
	}, b.input)
	if err != nil {
		return nil, nil, err
	}

	pkg := loadedPackages[0]

	rootMod := pkg.Module

	nextPaths := map[string]bool{}

	addPath := func(p string) {
		nextPaths[p] = true
	}

	addPath(rootMod.GoMod)

	var scan func(p *packages.Package)

	scan = func(p *packages.Package) {
		if strings.HasPrefix(p.PkgPath, rootMod.Path) {
			addPath(pkgDir(rootMod, p.PkgPath))

			for path := range p.Imports {
				scan(p.Imports[path])
			}
		}
	}

	scan(pkg)

	return pkg, nextPaths, err
}

func (b *WASMBuilder) registerPathsToWatch(nextPaths map[string]bool) {
	if b.watchedPaths != nil {
		b.watchedPaths = map[string]bool{}
	}

	for path := range nextPaths {
		if !b.watchedPaths[path] {
			if err := b.w.Add(path); err != nil {
				panic(err)
			}
		}
	}

	for path := range b.watchedPaths {
		if !nextPaths[path] {
			if err := b.w.Remove(path); err != nil {
				panic(err)
			}
		}
	}

	b.watchedPaths = nextPaths

}

func mustRel(base string, target string) string {
	r, _ := filepath.Rel(base, target)
	return r
}

func pkgDir(m *packages.Module, pkgPath string) string {
	return filepath.Join(m.Dir, func() string {
		return mustRel(m.Path, pkgPath)
	}())
}

var wasmExec, _ = os.ReadFile(filepath.Join(runtime.GOROOT(), "./misc/wasm/wasm_exec.js"))

func simplifyWasmExec(data []byte) []byte {
	start := bytes.Index(data, []byte("const enosys ="))
	end := bytes.Index(data, []byte(`})();`))

	data = data[start:end]

	data = bytes.ReplaceAll(data, []byte("global.Go = class"), []byte("class Go"))
	data = bytes.ReplaceAll(data, []byte(`typeof module !== "undefined"`), []byte("false"))
	data = bytes.ReplaceAll(data, []byte(`!global.TextDecoder`), []byte("false"))
	data = bytes.ReplaceAll(data, []byte(`!global.TextEncoder`), []byte("false"))
	data = bytes.ReplaceAll(data, []byte(`!global.performance`), []byte("false"))
	data = bytes.ReplaceAll(data, []byte(`!global.crypto`), []byte("false"))
	data = bytes.ReplaceAll(data, []byte(`!global.process`), []byte("false"))
	data = bytes.ReplaceAll(data, []byte(`!global.fs`), []byte("true"))
	data = bytes.ReplaceAll(data, []byte("global"), []byte("globalThis"))

	result := api.Transform(string(data), api.TransformOptions{
		Loader:            api.LoaderJSX,
		MinifySyntax:      true,
		MinifyIdentifiers: false,
		MinifyWhitespace:  false,
		LegalComments:     api.LegalCommentsNone,
	})

	return result.Code
}

func (b *WASMBuilder) pkgWASM(ctx context.Context, inputDir string) (err error) {
	outputWasm := filepath.Join(inputDir, "bin", "index"+".wasm")
	cwd, _ := os.Getwd()

	if err := run(
		ctx,
		[]string{"go", "build", "-o", outputWasm},
		append(os.Environ(), "GOOS=js", "GOARCH=wasm"),
		inputDir,
	); err != nil {
		return err
	}

	outputMjs := filepath.Join(filepath.Dir(outputWasm), "index.mjs")

	if err := os.WriteFile(
		outputMjs,
		[]byte(`import wasm from "./`+filepath.Base(outputWasm)+b.ImportWasmSuffix+`"               
`+string(simplifyWasmExec(wasmExec))+`                

export const main = async () => {
    const go = new Go();

    return WebAssembly
        .instantiateStreaming(
            fetch(wasm),
            go.importObject
        )
        .then((result) => go.run(result.instance));
}
`),
		os.ModePerm,
	); err != nil {
		return err
	}

	log := logr.FromContextOrDiscard(ctx)

	log.Info(fmt.Sprintf("%s/{%s,%s} generated.", filepath.Dir(mustRel(cwd, outputWasm)), filepath.Base(outputMjs), filepath.Base(outputWasm)))

	return nil
}

func run(ctx context.Context, command, env []string, dir string) error {
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Env = env
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return nil
}
