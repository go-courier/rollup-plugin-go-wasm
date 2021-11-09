# Rollup Plugin Go Wasm


## Features

```
app/
   main.go
   main.ts
```

```ts
// main.ts
// @ts-ignore
import ("main.go").then(({ main }) => main());
```

## In Vite

```js
import { defineConfig } from "vite";
import goWasm from "@go-courier/rollup-plugin-go-wasm";

export default defineConfig({
  root: "./app",
  build: {
    assetsDir: "static",
  },
  plugins: [
    goWasm({
      // don't added this if play with rollup
      importWasmSuffix: "?url",
    }),
  ],
});
```