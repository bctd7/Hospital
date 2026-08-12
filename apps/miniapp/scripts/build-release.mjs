import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const appRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const releaseConfigPath = join(appRoot, "release.config.json");
const releaseConfig = JSON.parse(readFileSync(releaseConfigPath, "utf8"));
if (releaseConfig.apiTransport !== "cloudbase") {
  throw new Error("release transport must be cloudbase");
}
for (const key of ["cloudBaseEnvId", "anyServiceName"]) {
  if (typeof releaseConfig[key] !== "string" || !releaseConfig[key].trim()) {
    throw new Error(`${key} is required for release builds`);
  }
}

const releaseEnv = {
  VITE_API_TRANSPORT: releaseConfig.apiTransport,
  VITE_CLOUDBASE_ENV_ID: releaseConfig.cloudBaseEnvId.trim(),
  VITE_ANYSERVICE_NAME: releaseConfig.anyServiceName.trim(),
};

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
  releaseConfig.cloudBaseEnvId.trim(),
  releaseConfig.anyServiceName.trim(),
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
