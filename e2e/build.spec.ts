import { expect, test } from "@playwright/test";
import { TestApiClient } from "./fixtures";

// Run only against the isolated local server with the deterministic LLM fixture.
// This exercises real HTTP, queue, database, compiler and UI, not model quality.
test("build planning clarification, free text, cancellation and saved result", async ({
  page,
  baseURL,
}, testInfo) => {
  test.skip(
    process.env.CHIMII_BUILD_E2E_STUB !== "1",
    "requires an isolated server with the Build LLM fixture",
  );
  test.setTimeout(180_000);
  const api = new TestApiClient();
  await api.login(`build-${Date.now()}@example.test`, "Build browser test");
  const workspace = await api.ensureWorkspace(
    "Build browser test",
    `build-${Date.now()}`,
  );
  await api.markUserOnboarded();
  const errors: string[] = [];
  const electronicRequests: string[] = [];
  page.on("request", (request) => {
    if (/\/api\/circuit\/(kits|catalog|inventory)/.test(request.url()))
      electronicRequests.push(request.url());
  });
  page.on("pageerror", (error) => errors.push(error.message));
  try {
    await page.context().addCookies(
      api.getAuthCookies().map((cookie) => ({
        ...cookie,
        url: baseURL!,
        httpOnly: cookie.name === "chimii_auth",
        sameSite: "Lax" as const,
      })),
    );
    await page
      .context()
      .addCookies([{ name: "chimii-locale", value: "zh-Hans", url: baseURL! }]);
    await page.addInitScript(() => {
      localStorage.setItem("chimii_locale", "zh-Hans");
      localStorage.setItem("chimii:chat:isOpen", "false");
    });
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.goto(`/${workspace.slug}/build`, {
      waitUntil: "domcontentloaded",
    });
    const idea = page.getByRole("textbox", { name: "我的发明想法" });
    await expect(idea).toBeVisible({ timeout: 60_000 });
    await expect(
      page.getByRole("heading", { name: "积木创作" }),
    ).toBeVisible();
    await expect(page.getByRole("combobox")).toHaveCount(0);
    await page.screenshot({
      path: testInfo.outputPath("brick-home-desktop.png"),
      fullPage: true,
    });
    await idea.fill("我想做一个积木朋友");
    await page.getByRole("button", { name: "开始创造", exact: true }).click();
    await expect(
      page.getByRole("heading", {
        name: "你想做怎样的积木朋友？",
        exact: true,
      }),
    ).toBeVisible({ timeout: 30_000 });
    await page.screenshot({
      path: testInfo.outputPath("clarification-desktop.png"),
      fullPage: true,
    });
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(page.locator("#build-answer")).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: testInfo.outputPath("clarification-mobile.png"),
      fullPage: true,
    });
    await page.getByRole("button", { name: "修改想法", exact: true }).click();
    await expect(idea).toBeEnabled();
    await idea.fill("我想做一个积木朋友");
    await page.getByRole("button", { name: "开始创造", exact: true }).click();
    await expect(
      page.getByRole("heading", {
        name: "你想做怎样的积木朋友？",
        exact: true,
      }),
    ).toBeVisible();
    await page.locator("#build-answer").fill("一个站着的机器人");
    await page.getByRole("button", { name: "继续创造", exact: true }).click();
    await expect(page.getByText("搭建验证通过", { exact: true })).toBeVisible({
      timeout: 30_000,
    });
    await page.screenshot({
      path: testInfo.outputPath("saved-build.png"),
      fullPage: true,
    });
    let fullListRequests = 0;
    page.on("request", (r) => {
      if (
        new URL(r.url()).pathname === "/api/build/creations" &&
        !r.url().includes("view=summary")
      )
        fullListRequests++;
    });
    await page.getByRole("button", { name: "开始搭建", exact: true }).click();
    await expect(
      page.getByText("已保存到第 1 步", { exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: "下一步", exact: true }).click();
    await expect(
      page.getByText("已保存到第 2 步", { exact: true }),
    ).toBeVisible();
    await page.goto(`/${workspace.slug}/creations`, {
      waitUntil: "domcontentloaded",
    });
    await page.getByRole("link", { name: /积木机器人/ }).click();
    await expect(page).toHaveURL(
      new RegExp(`/${workspace.slug}/creations/[^/]+$`),
    );
    await page.reload();
    await expect(
      page.getByText("已保存到第 2 步", { exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: "继续搭建", exact: true }).click();
    await expect(
      page.getByText("已保存到第 2 步", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "下一步", exact: true }),
    ).toBeEnabled();
    expect(fullListRequests).toBe(0);
    await page.screenshot({
      path: testInfo.outputPath("build-progress-mobile.png"),
      fullPage: true,
    });

    await page.goto(`/${workspace.slug}/build`, {
      waitUntil: "domcontentloaded",
    });
    await expect(
      page.getByRole("heading", { name: "积木创作" }),
    ).toBeVisible();
    await expect(page.getByRole("combobox")).toHaveCount(0);
    await page.screenshot({
      path: testInfo.outputPath("brick-home-mobile.png"),
      fullPage: true,
    });
    await idea.fill("做一个能真的飞起来的飞机");
    await page.getByRole("button", { name: "开始创造", exact: true }).click();
    await expect(
      page.getByText("当前还不能制作真正飞起来的机构。", { exact: true }).filter({ visible: true }),
    ).toBeVisible({ timeout: 30_000 });
    expect(errors).toEqual([]);
    expect(electronicRequests).toEqual([]);
  } finally {
    await api.deleteWorkspace();
    await api.cleanup();
  }
});
