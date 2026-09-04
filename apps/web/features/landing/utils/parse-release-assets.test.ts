import { describe, expect, it } from "vitest";
import {
  hasCompleteDesktopAssetSet,
  parseReleaseAssets,
} from "./parse-release-assets";

function asset(name: string) {
  return {
    name,
    browser_download_url: `https://github.test/releases/${name}`,
  };
}

describe("parseReleaseAssets", () => {
  it("keeps both Apple Silicon and Intel macOS installers", () => {
    const assets = parseReleaseAssets([
      asset("chimii-desktop-0.4.2-mac-arm64.dmg"),
      asset("chimii-desktop-0.4.2-mac-arm64.zip"),
      asset("chimii-desktop-0.4.2-mac-x64.dmg"),
      asset("chimii-desktop-0.4.2-mac-x64.zip"),
      asset("chimii-desktop-0.4.2-mac-x64.dmg.blockmap"),
      asset("latest-x64-mac.yml"),
    ]);

    expect(assets).toEqual({
      macArm64Dmg:
        "https://github.test/releases/chimii-desktop-0.4.2-mac-arm64.dmg",
      macArm64Zip:
        "https://github.test/releases/chimii-desktop-0.4.2-mac-arm64.zip",
      macX64Dmg:
        "https://github.test/releases/chimii-desktop-0.4.2-mac-x64.dmg",
      macX64Zip:
        "https://github.test/releases/chimii-desktop-0.4.2-mac-x64.zip",
    });
  });

  it("recognizes the complete electron-builder platform matrix", () => {
    const names = [
      "chimii-desktop-0.2.6-mac-arm64.dmg",
      "chimii-desktop-0.2.6-mac-arm64.zip",
      "chimii-desktop-0.2.6-mac-x64.dmg",
      "chimii-desktop-0.2.6-mac-x64.zip",
      "chimii-desktop-0.2.6-windows-x64.exe",
      "chimii-desktop-0.2.6-windows-arm64.exe",
      "chimii-desktop-0.2.6-linux-x86_64.AppImage",
      "chimii-desktop-0.2.6-linux-amd64.deb",
      "chimii-desktop-0.2.6-linux-x86_64.rpm",
      "chimii-desktop-0.2.6-linux-arm64.AppImage",
      "chimii-desktop-0.2.6-linux-arm64.deb",
      "chimii-desktop-0.2.6-linux-arm64.rpm",
    ];

    const parsed = parseReleaseAssets(names.map(asset));

    expect(hasCompleteDesktopAssetSet(parsed)).toBe(true);
    expect(parsed.linuxAmd64AppImage).toContain("linux-x86_64.AppImage");
    expect(parsed.linuxArm64Rpm).toContain("linux-arm64.rpm");
  });
});
