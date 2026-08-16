import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
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
const developmentURL = new URL(developmentEnv.VITE_API_BASE_URL.trim());
if (
  developmentURL.protocol !== "http:" ||
  !["127.0.0.1", "localhost"].includes(developmentURL.hostname)
) {
  throw new Error(
    "development API must use http://127.0.0.1 or http://localhost; deploy the backend before testing on a phone",
  );
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
const requiredOutputPaths = [
  join(outputRoot, "app.js"),
  join(outputRoot, "app.json"),
  join(outputRoot, "app.wxss"),
  join(outputRoot, "project.config.json"),
];
const buildStartedAt = Date.now();
let completed = false;
let buildSignaled = false;
let terminating = false;
const timeout = setTimeout(() => {
  if (completed) return;
  terminating = true;
  child.kill();
  console.error("Development miniapp build timed out.");
}, 120_000);

function outputFiles(directory) {
  if (!existsSync(directory)) return [];
  return readdirSync(directory, { withFileTypes: true })
    .flatMap((entry) => {
      const path = join(directory, entry.name);
      return entry.isDirectory() ? outputFiles(path) : [path];
    })
    .sort();
}

function validJsonOutput(files) {
  try {
    for (const path of files.filter((value) => value.endsWith(".json"))) {
      JSON.parse(readFileSync(path, "utf8"));
    }
    return true;
  } catch {
    return false;
  }
}

function completeOutputState() {
  if (
    !existsSync(environmentPath) ||
    !existsSync(clientPath) ||
    requiredOutputPaths.some((path) => !existsSync(path) || statSync(path).size === 0)
  ) {
    return "";
  }

  const files = outputFiles(outputRoot);
  if (!files.length || !validJsonOutput(files)) return "";
  const environmentOutput = readFileSync(environmentPath, "utf8");
  const clientOutput = readFileSync(clientPath, "utf8");
  if (
    !environmentOutput.includes(developmentEnv.VITE_API_BASE_URL.trim()) ||
    !clientOutput.includes("directRequest") ||
    !files.some((path) => statSync(path).mtimeMs >= buildStartedAt)
  ) {
    return "";
  }

  return files
    .map((path) => {
      const stats = statSync(path);
      return `${path}:${stats.size}:${stats.mtimeMs}`;
    })
    .join("\n");
}

function forward(chunk, destination) {
  const value = chunk.toString();
  destination.write(value);
  if (!buildSignaled && value.includes("DONE  Build complete.")) {
    buildSignaled = true;
    const outputDeadline = Date.now() + 10_000;
    let lastOutputState = "";
    let stableSince = 0;
    const outputPoll = setInterval(() => {
      const outputState = completeOutputState();
      if (outputState && outputState === lastOutputState) {
        if (stableSince === 0) stableSince = Date.now();
        if (Date.now() - stableSince >= 1_000) {
          clearInterval(outputPoll);
          completed = true;
          terminating = true;
          child.kill();
          return;
        }
      } else {
        lastOutputState = outputState;
        stableSince = 0;
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
