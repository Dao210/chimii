import { afterEach, describe, expect, it, vi } from "vitest";
import { fetchLatestRelease } from "./github-release";

const assetNames = (version: string) => [
  `chimii-desktop-${version}-mac-arm64.dmg`,
  `chimii-desktop-${version}-mac-arm64.zip`,
  `chimii-desktop-${version}-mac-x64.dmg`,
  `chimii-desktop-${version}-mac-x64.zip`,
  `chimii-desktop-${version}-windows-x64.exe`,
  `chimii-desktop-${version}-windows-arm64.exe`,
  `chimii-desktop-${version}-linux-x86_64.AppImage`,
  `chimii-desktop-${version}-linux-amd64.deb`,
  `chimii-desktop-${version}-linux-x86_64.rpm`,
  `chimii-desktop-${version}-linux-arm64.AppImage`,
  `chimii-desktop-${version}-linux-arm64.deb`,
  `chimii-desktop-${version}-linux-aarch64.rpm`,
];

function asset(name: string) {
  return {
    name,
    browser_download_url: `https://github.com/Dao210/chimii/releases/download/v-test/${name}`,
  };
}

function completeAssets(version: string) {
  return assetNames(version).map(asset);
}

function releasePayload(overrides: {
  tag: string;
  publishedMinutesAgo?: number;
  assets?: Array<{ name: string; browser_download_url: string }>;
  prerelease?: boolean;
  draft?: boolean;
}) {
  const published = new Date(
    Date.now() - (overrides.publishedMinutesAgo ?? 0) * 60_000,
  ).toISOString();
  return {
    tag_name: overrides.tag,
    published_at: published,
    html_url: `https://github.com/Dao210/chimii/releases/tag/${overrides.tag}`,
    prerelease: overrides.prerelease ?? false,
    draft: overrides.draft ?? false,
    assets: overrides.assets ?? [],
  };
}

function mockFetchWithReleases(releases: unknown[]) {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(releases), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("fetchLatestRelease", () => {
  it("uses a complete previous release while a fresh release is incomplete", async () => {
    const latestAsset = asset("chimii-desktop-0.2.14-windows-x64.exe");
    const previousAssets = completeAssets("0.2.13");
    mockFetchWithReleases([
      releasePayload({
        tag: "v0.2.14",
        publishedMinutesAgo: 10,
        assets: [latestAsset],
      }),
      releasePayload({
        tag: "v0.2.13",
        publishedMinutesAgo: 60 * 24,
        assets: previousAssets,
      }),
    ]);

    const result = await fetchLatestRelease();
    expect(result.version).toBe("v0.2.13");
    expect(result.status).toBe("ready");
    expect(result.assets.macArm64Dmg).toBe(previousAssets[0]?.browser_download_url);
  });

  it("uses a complete latest release immediately", async () => {
    const latestAssets = completeAssets("0.2.14");
    const fetchMock = mockFetchWithReleases([
      releasePayload({
        tag: "v0.2.14",
        publishedMinutesAgo: 10,
        assets: latestAssets,
      }),
      releasePayload({
        tag: "v0.2.13",
        publishedMinutesAgo: 60 * 24,
        assets: completeAssets("0.2.13"),
      }),
    ]);

    const result = await fetchLatestRelease();
    expect(result.version).toBe("v0.2.14");
    expect(result.status).toBe("ready");
    expect(result.assets.macArm64Dmg).toBe(latestAssets[0]?.browser_download_url);
    expect(fetchMock).toHaveBeenCalledWith(
      "https://api.github.com/repos/Dao210/chimii/releases?per_page=2",
      expect.objectContaining({
        headers: expect.objectContaining({
          Accept: "application/vnd.github+json",
        }),
      }),
    );
  });

  it("surfaces an incomplete first release instead of hiding its assets", async () => {
    const firstAsset = asset("chimii-desktop-0.2.14-windows-x64.exe");
    mockFetchWithReleases([
      releasePayload({
        tag: "v0.0.1",
        publishedMinutesAgo: 5,
        assets: [firstAsset],
      }),
    ]);

    const result = await fetchLatestRelease();
    expect(result.version).toBe("v0.0.1");
    expect(result.status).toBe("incomplete");
    expect(result.assets.winX64Exe).toBe(firstAsset.browser_download_url);
  });

  it("skips prereleases and drafts in the candidate list", async () => {
    mockFetchWithReleases([
      releasePayload({
        tag: "v0.2.15-rc.1",
        publishedMinutesAgo: 30,
        prerelease: true,
      }),
      releasePayload({
        tag: "v0.2.14",
        publishedMinutesAgo: 120,
        assets: completeAssets("0.2.14"),
      }),
    ]);

    const result = await fetchLatestRelease();
    expect(result.version).toBe("v0.2.14");
  });

  it("returns an empty release shape when the API errors", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("rate limited", { status: 403 }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

    const result = await fetchLatestRelease();
    expect(result).toEqual({
      version: null,
      publishedAt: null,
      htmlUrl: null,
      assets: {},
      status: "source_unavailable",
    });
    expect(warnSpy).toHaveBeenCalled();
    warnSpy.mockRestore();
  });

  it("returns an empty release shape when all candidates are filtered out", async () => {
    mockFetchWithReleases([
      releasePayload({ tag: "v0.2.15-rc.1", prerelease: true }),
      releasePayload({ tag: "v0.2.14-draft", draft: true }),
    ]);

    const result = await fetchLatestRelease();
    expect(result.version).toBeNull();
    expect(result.assets).toEqual({});
    expect(result.status).toBe("no_release");
  });
});
