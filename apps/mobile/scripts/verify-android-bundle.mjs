import { execFileSync } from "node:child_process";
import assert from "node:assert/strict";

const apk = process.argv[2];
assert(apk, "APK path is required");
const bundle = execFileSync("unzip", ["-p", apk, "assets/index.android.bundle"], {
  maxBuffer: 64 * 1024 * 1024,
});
assert.equal(bundle.subarray(0, 8).toString("hex"), "c61fbc03c103191f", "Missing Hermes bytecode");
for (const name of ["EXPO_PUBLIC_API_URL", "EXPO_PUBLIC_WEB_URL"]) {
  const value = process.env[name];
  assert(value, `${name} is required`);
  if (process.env.APP_ENV === "production") assert.equal(new URL(value).protocol, "https:");
  assert(bundle.includes(Buffer.from(value)), `APK does not contain configured ${name}`);
}
if (process.env.APP_ENV === "production") {
  for (const value of ["http://10.0.2.2:", "local-fixture-token"])
    assert(!bundle.includes(Buffer.from(value)), "APK contains local acceptance configuration");
}
console.log("APK Hermes bundle and service environment verified");
