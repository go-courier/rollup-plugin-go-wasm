import { InputOptions, Plugin, ResolveIdResult } from "rollup";
import { dirname, join } from "path";
import * as console from "console";
import { ChildProcess, spawn, spawnSync } from "child_process";

export interface GoWasmPackOption {
  importWasmSuffix?: string;
  watch?: boolean;
}

const toKebabCase = (k: string): string => {
  return k.split("").map((c) => c >= "A" && c <= "Z" ? "-" + c.toLowerCase() : c).join("");
};

const optionsToFlags = (options: { [k: string]: string }) => {
  const flags: string[] = [];

  Object.keys(options).forEach((k) => {
    const v = (options as any)[k] || "";
    if (v) {
      flags.push(`--${toKebabCase(k)}=${v}`);
    }
  });

  return flags;
};

export default function goWasmPack(options: GoWasmPackOption): Plugin & { config: (...args: any) => any } {
  let watchMode = false;

  const tasks: { [path: string]: ChildProcess } = {};

  const prebuild = async (mainRoot: string) => {
    if (tasks[mainRoot]) {
      return Promise.resolve();
    }

    tasks[mainRoot] = spawn("go-wasm-pack", [...optionsToFlags({
      importWasmSuffix: options.importWasmSuffix || "",
      watch: `${watchMode}`,
    }), mainRoot]);

    let compiled = false;

    return new Promise((resolve, reject) => {
        tasks[mainRoot].stderr!.on("data", (data) => {
          console.log(data.toString());
        });

        tasks[mainRoot].stdout!.on("data", (data) => {
          const l = data.toString();

          if (l) {
            console.log(l);

            if (!compiled && l.indexOf("generated") > -1) {
              resolve(undefined);
              compiled = true;
            }
          }
        });
      },
    );
  };


  return {
    name: "@go-courier/rollup-plugin-go-wasm",

    // vite only
    config(_: any, { command }: any) {
      if (command === "serve") {
        watchMode = true;
      }
    },

    options(o) {
      if (o.watch) {
        watchMode = true;
      }
      return o;
    },

    async resolveId(id: string, importer?: string): Promise<ResolveIdResult> {
      if (id === "main.go") {
        await prebuild(dirname(importer!));
        return join(dirname(importer!), "./bin/index.mjs");
      }
      return;
    },
  };
}
