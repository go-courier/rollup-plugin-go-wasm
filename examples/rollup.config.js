import goWasm from "..";
import url from "@rollup/plugin-url";

export default {
  output: {
    dir: "./dist",
  },
  input: "./app/main.js",
  plugins: [
    goWasm({
      // importWasmSuffix: "?url",
    }),
    url({
      include: ["**/*.wasm"],
    }),
  ],
};