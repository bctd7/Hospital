import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const appRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const productionEnvPath = join(appRoot, ".env.production");

function parseEnv(source) {
  const values = {};
  for (const rawLine of source.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    const separator = line.indexOf("=");
    if (separator < 1) continue;
    const key = line.slice(0, separator).trim();
    let value = line.slice(separator + 1).trim();
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }
    values[key] = value;
  }
  return values;
}

const releaseEnv = parseEnv(readFileSync(productionEnvPath, "utf8"));
if (releaseEnv.VITE_API_TRANSPORT !== "cloudbase") {
  throw new Error("release transport must be cloudbase");
}
for (const key of ["VITE_CLOUDBASE_ENV_ID", "VITE_ANYSERVICE_NAME"]) {
  if (!releaseEnv[key]) throw new Error(`${key} is required for release builds`);
}

const childEnv = { ...process.env };
for (const key of Object.keys(childEnv)) {
  if (key.startsWith("VITE_")) delete childEnv[key];
}
Object.assign(childEnv, releaseEnv, {
  // Prevent ignored local env files from injecting a LAN endpoint. CloudBase
  // transport does not use a direct base URL.
  VITE_API_BASE_URL: "",
});

const uni = join(
  appRoot,
  "node_modules",
  "@dcloudio",
  "vite-plugin-uni",
  "bin",
  "uni.js",
);
const result = spawnSync(process.execPath, [uni, "build", "-p", "mp-weixin"], {
  cwd: appRoot,
  env: childEnv,
  stdio: "inherit",
});
if (result.error) throw result.error;
if (result.status !== 0) process.exit(result.status ?? 1);

const outputRoot = join(appRoot, "dist", "build", "mp-weixin");
const environmentOutput = readFileSync(
  join(outputRoot, "config", "environment.js"),
  "utf8",
);
const clientOutput = readFileSync(join(outputRoot, "api", "client.js"), "utf8");
for (const expected of [
  releaseEnv.VITE_CLOUDBASE_ENV_ID,
  releaseEnv.VITE_ANYSERVICE_NAME,
]) {
  if (!environmentOutput.includes(expected)) {
    throw new Error(`release output is missing CloudBase value: ${expected}`);
  }
}
if (!clientOutput.includes("callAnyService")) {
  throw new Error("release output is not using CloudBase AnyService transport");
}
if (
  /(?:10\.|192\.168\.|172\.(?:1[6-9]|2\d|3[01])\.)\d{1,3}(?:\.\d{1,3}){2}/.test(
    environmentOutput,
  )
) {
  throw new Error("release output contains a private-network API address");
}

console.log("Release transport verified: CloudBase AnyService");
