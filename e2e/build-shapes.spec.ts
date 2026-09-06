import { expect, test } from "@playwright/test";
import { TestApiClient } from "./fixtures";

if (process.env.CHIMII_BUILD_WEBGL_TEST === "1") {
  test.use({
    channel: "chrome",
    launchOptions: {
      args: process.platform === "darwin" ? ["--use-angle=metal"] : [],
    },
  });
}

test("generic clock generation, editable revision and retained original", async ({
  page,
  baseURL,
}, testInfo) => {
  test.skip(
    process.env.CHIMII_BUILD_E2E_STUB !== "1",
    "requires isolated Build fixture",
  );
  test.setTimeout(180_000);
  const api = new TestApiClient();
  await api.login(`shape-${Date.now()}@example.test`, "Shape browser test");
  const workspace = await api.ensureWorkspace(
    "Shape browser test",
    `shape-${Date.now()}`,
  );
  await api.markUserOnboarded();
  const errors: string[] = [];
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
    await page
      .getByRole("textbox", { name: "我的发明想法" })
      .fill("做一个有钟面、两根静态指针和刻度的时钟造型");
    await page.getByRole("button", { name: "开始创造", exact: true }).click();
    await expect(page.getByText("搭建验证通过", { exact: true })).toBeVisible({
      timeout: 40_000,
    });
    await expect(
      page.getByRole("button", { name: "修改造型", exact: true }),
    ).toBeVisible();
    if (process.env.CHIMII_BUILD_WEBGL_TEST === "1") {
      await expect(page.locator('[data-renderer="ldraw-glb"]')).toBeVisible({
        timeout: 20_000,
      });
      await expect(page.locator('[data-renderer="ldraw-glb"] > svg')).toHaveCSS(
        "opacity",
        "0",
      );
    }
    await page.screenshot({
      path: testInfo.outputPath("clock-generated.png"),
      fullPage: true,
    });
    await page.getByRole("button", { name: "修改造型", exact: true }).click();
    const editor = page.getByRole("region", { name: "修改造型" });
    await expect(editor.getByLabel("选择形状")).toHaveValue("dial");
    await editor.getByLabel("颜色", { exact: true }).selectOption("2");
    await editor.getByRole("button", { name: "撤销修改" }).click();
    await expect(editor.getByLabel("颜色", { exact: true })).toHaveValue("15");
    await editor.getByRole("button", { name: "重做修改" }).click();
    await expect(editor.getByLabel("颜色", { exact: true })).toHaveValue("2");
    await page.setViewportSize({ width: 390, height: 844 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: testInfo.outputPath("clock-editor-mobile.png"),
      fullPage: true,
    });
    await editor
      .getByRole("button", { name: "生成修改版", exact: true })
      .click();
    await expect(editor.getByRole("link", { name: "打开修改版" })).toBeVisible({
      timeout: 40_000,
    });
    await editor.getByRole("link", { name: "打开修改版" }).click();
    await expect(page).toHaveURL(
      new RegExp(`/${workspace.slug}/creations/[^/]+$`),
    );
    await expect(page.getByText("搭建验证通过", { exact: true })).toBeVisible();
    await page.reload();
    await page.getByRole("button", { name: "修改造型", exact: true }).click();
    await expect(page.getByLabel("颜色", { exact: true })).toHaveValue("2");
    await page.screenshot({
      path: testInfo.outputPath("clock-revision.png"),
      fullPage: true,
    });
    await page.getByLabel("用一句话修改").fill("把钟面换一个颜色");
    await page.getByRole("button", { name: "按描述修改", exact: true }).click();
    await expect(page.getByText("钟面想换成什么颜色？")).toBeVisible({
      timeout: 30_000,
    });
    await page.getByRole("button", { name: "红色", exact: true }).click();
    await expect(page.getByRole("link", { name: "打开修改版" })).toBeVisible({
      timeout: 40_000,
    });
    const describedRevisionURL = await page
      .getByRole("link", { name: "打开修改版" })
      .getAttribute("href");
    await page.getByRole("link", { name: "打开修改版" }).click();
    await expect(page).toHaveURL(new URL(describedRevisionURL!, baseURL!).href);
    await page.getByRole("button", { name: "修改造型", exact: true }).click();
    await expect(page.getByLabel("颜色", { exact: true })).toHaveValue("4");
    expect(errors).toEqual([]);
  } finally {
    await api.deleteWorkspace();
    await api.cleanup();
  }
});
