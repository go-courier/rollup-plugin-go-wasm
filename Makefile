debug.vite: build
	rm -rf ./examples/app/bin
	cd ./examples && pnpx vite

debug.watch: build
	rm -rf ./examples/app/bin
	cd ./examples && pnpx rollup --watch -c rollup.config.js

debug: build
	rm -rf ./examples/app/bin
	cd ./examples && pnpx rollup -c rollup.config.js

d:
	go run ./go-wasm-pack ./examples/app
	go run ./go-wasm-pack --list=true ./examples/app | jq
	go run ./go-wasm-pack --watch ./examples/app

build: npm.install
	rm -rf dist/
	pnpx tsc -p .

fmt:
	goimports -l -w ./wasm-pack
	pnpx prettier -w ./src

npm.install:
	pnpm install

publish: build
	npm publish
