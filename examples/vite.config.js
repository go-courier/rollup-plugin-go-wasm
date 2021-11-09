import { defineConfig } from "vite";
import goWasm from "..";

export default defineConfig({
  root: "./app",
  build: {
    assetsDir: "static",
  },
  plugins: [
    goWasm({
      importWasmSuffix: "?url",
    }),
  ],
});