import { readFile, writeFile } from "node:fs/promises";
import { expect, test } from "@playwright/test";
import { TestApiClient } from "./fixtures";

// Explicit software WebGL is required by current headless Chromium.
test.use({ launchOptions: { args: ["--enable-unsafe-swiftshader"] } });

// Supply a catalog-verified CompileResult. This exercises persisted rendering
// and workflow; an authored fixture does not establish model-generation quality.
test("compiled complex scene renders and exports its full BOM", async ({ page, baseURL }, testInfo) => {
  test.skip(!process.env.CHIMII_SCENE_FIXTURE_JSON, "requires an explicitly compiled local scene fixture");
  test.setTimeout(180_000);
  const result = JSON.parse(await readFile(process.env.CHIMII_SCENE_FIXTURE_JSON!, "utf8"));
  const api = new TestApiClient();
  await api.login(`scene-${Date.now()}@example.test`, "Scene acceptance");
  const workspace = await api.ensureWorkspace("Scene acceptance", `scene-${Date.now()}`);
  await api.markUserOnboarded();
  const id = await api.seedCompiledBrickCreation(result);
  const errors: string[] = [];
  const logs: string[] = [];
  page.on("console", message => { if (["error", "warning"].includes(message.type())) logs.push(message.text()); });
  page.on("requestfailed", request => logs.push(`${request.url()}: ${request.failure()?.errorText}`));
  page.on("pageerror", error => errors.push(error.message));
  try {
    await page.context().addCookies(api.getAuthCookies().map(cookie => ({ ...cookie, url: baseURL!, httpOnly: cookie.name === "chimii_auth", sameSite: "Lax" as const })));
    await page.context().addCookies([{ name: "chimii-locale", value: "zh-Hans", url: baseURL! }]);
    await page.addInitScript(() => { localStorage.setItem("chimii_locale", "zh-Hans"); localStorage.setItem("chimii:chat:isOpen", "false"); });
    await page.setViewportSize({ width: 1440, height: 1100 });
    await page.goto(`/${workspace.slug}/creations/${id}`, { waitUntil: "domcontentloaded" });
    await expect(page.locator('[data-renderer="ldraw-glb"]')).toBeVisible({ timeout: 90_000 });
    await expect(page.getByText("设计说明", { exact: true })).toBeVisible();
    await expect(page.getByText(/尚未实物试搭/)).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath("complex-scene-desktop.png"), fullPage: true });
    await page.getByText(/零件清单 ·/).click();
    const download = page.waitForEvent("download");
    await page.getByRole("button", { name: "下载零件清单", exact: true }).click();
    const file = await download;
    const csv = await readFile((await file.path())!, "utf8");
    const count = csv.trim().split("\r\n").slice(1).reduce((total, line) => total + Number(line.split(",")[4]!.replaceAll('"', "")), 0);
    expect(count).toBe(result.plan.placements.length);
    await page.getByRole("button", { name: "开始搭建", exact: true }).click();
    await expect(page.getByText("已保存到第 1 步", { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "下一步", exact: true }).click();
    await expect(page.getByText("已保存到第 2 步", { exact: true })).toBeVisible();
    const inspectStep = Number(process.env.CHIMII_SCENE_INSPECT_STEP ?? 0);
    if (inspectStep > 2 && inspectStep <= result.plan.steps.length && Number.isInteger(inspectStep)) {
      for (let step = 3; step <= inspectStep; step++) {
        await page.getByRole("button", { name: "下一步", exact: true }).click();
        await expect(page.getByText(`已保存到第 ${step} 步`, { exact: true })).toBeVisible();
      }
      await page.screenshot({ path: testInfo.outputPath("construction-detail.png"), fullPage: true });
    }
    await page.setViewportSize({ width: 390, height: 844 });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await page.screenshot({ path: testInfo.outputPath("complex-scene-mobile.png"), fullPage: true });
    expect(errors).toEqual([]);
  } finally {
    await writeFile(testInfo.outputPath("preview-diagnostics.json"), JSON.stringify({ errors, logs }, null, 2));
    await page.screenshot({ path: testInfo.outputPath("scene-final-state.png"), fullPage: true });
    await api.deleteWorkspace();
  }
});
