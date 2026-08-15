import { existsSync, readFileSync } from "node:fs";
import { spawn } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { loadEnv } from "vite";

const appRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const developmentEnv = loadEnv("development", appRoot, "VITE_");
if (developmentEnv.VITE_API_TRANSPORT !== "direct") {
  throw new Error("development transport must be direct");
}
if (!developmentEnv.VITE_API_BASE_URL?.trim()) {
  throw new Error("VITE_API_BASE_URL is required for development builds");
}

const uni = join(
  appRoot,
  "node_modules",
  "@dcloudio",
  "vite-plugin-uni",
  "bin",
  "uni.js",
);
const childEnv = { ...process.env };
// PowerShell or a previous release build may leave VITE_* variables in the
// parent process. Process variables take precedence over .env.local, so clear
// them before applying the development configuration explicitly.
for (const key of Object.keys(childEnv)) {
  if (key.startsWith("VITE_")) delete childEnv[key];
}
Object.assign(childEnv, developmentEnv, {
  NODE_ENV: "development",
  VITE_API_TRANSPORT: "direct",
  VITE_API_BASE_URL: developmentEnv.VITE_API_BASE_URL.trim(),
});

const child = spawn(
  process.execPath,
  [uni, "build", "-p", "mp-weixin", "--watch"],
  {
    cwd: appRoot,
    env: childEnv,
    stdio: ["ignore", "pipe", "pipe"],
  },
);

const outputRoot = join(appRoot, "dist", "dev", "mp-weixin");
const environmentPath = join(outputRoot, "config", "environment.js");
const clientPath = join(outputRoot, "api", "client.js");
let completed = false;
let buildSignaled = false;
let terminating = false;
const timeout = setTimeout(() => {
  if (completed) return;
  terminating = true;
  child.kill();
  console.error("Development miniapp build timed out.");
}, 120_000);

function forward(chunk, destination) {
  const value = chunk.toString();
  destination.write(value);
  if (!buildSignaled && value.includes("DONE  Build complete.")) {
    buildSignaled = true;
    const outputDeadline = Date.now() + 10_000;
    const outputPoll = setInterval(() => {
      if (existsSync(environmentPath) && existsSync(clientPath)) {
        const environmentOutput = readFileSync(environmentPath, "utf8");
        const clientOutput = readFileSync(clientPath, "utf8");
        const outputIsComplete =
          environmentOutput.includes(developmentEnv.VITE_API_BASE_URL.trim()) &&
          clientOutput.includes("directRequest");
        if (outputIsComplete) {
          clearInterval(outputPoll);
          completed = true;
          terminating = true;
          child.kill();
          return;
        }
      }
      if (Date.now() >= outputDeadline) {
        clearInterval(outputPoll);
        terminating = true;
        child.kill();
      }
    }, 100);
  }
}

child.stdout.on("data", (chunk) => forward(chunk, process.stdout));
child.stderr.on("data", (chunk) => forward(chunk, process.stderr));
child.on("error", (error) => {
  clearTimeout(timeout);
  throw error;
});
child.on("exit", (code) => {
  clearTimeout(timeout);
  if (!completed) {
    process.exitCode = terminating ? 1 : code ?? 1;
    return;
  }

  const environmentOutput = readFileSync(
    environmentPath,
    "utf8",
  );
  const clientOutput = readFileSync(clientPath, "utf8");
  if (
    !environmentOutput.includes(developmentEnv.VITE_API_BASE_URL.trim()) ||
    !clientOutput.includes("directRequest")
  ) {
    throw new Error("development output is not using direct API transport");
  }
  console.log("Development transport verified: direct API request");
});
