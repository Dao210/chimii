import { expect, test } from "@playwright/test";
import { TestApiClient } from "./fixtures";

test("browses source-backed module assembly without creating a design or changing stock", async ({ page, baseURL }, testInfo) => {
  test.setTimeout(120_000);
  const api = new TestApiClient();
  await api.login(`assembly-reference-${Date.now()}@example.test`, "Assembly reference test");
  const ws = await api.ensureWorkspace("Assembly reference test", `assembly-reference-${Date.now()}`);
  await api.markUserOnboarded();
  const errors: string[] = [];
  const writes: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => {
    if (/\/api\/(circuit|build\/sessions|build\/conversations)/.test(request.url()) && ["POST", "PUT", "PATCH", "DELETE"].includes(request.method())) writes.push(request.url());
  });
  try {
    await page.context().addCookies(api.getAuthCookies().map((cookie) => ({
      ...cookie, url: baseURL!, httpOnly: cookie.name === "chimii_auth", sameSite: "Lax" as const,
    })));
    await page.addInitScript(() => {
      localStorage.setItem("chimii:chat:isOpen", "false");
      localStorage.setItem("chimii_locale", "zh-Hans");
    });
    const headers = { Authorization: `Bearer ${api.getToken()}`, "X-Workspace-Slug": ws.slug };
    const csrf = api.getAuthCookies().find((cookie) => cookie.name === "chimii_csrf")!.value;
    expect((await page.request.patch("/api/me", {
      headers: { ...headers, "X-CSRF-Token": csrf }, data: { language: "zh-Hans" },
    })).ok()).toBe(true);
    await page.context().addCookies([{ name: "chimii-locale", value: "zh-Hans", url: baseURL!, sameSite: "Lax" }]);
    const response = await page.request.get("/api/circuit/kits", { headers });
    expect(response.ok()).toBe(true);
    const payload = await response.json();
    expect(payload.kits.map((kit: { kit_id: string }) => kit.kit_id)).not.toContain("elecfreaks-ef08288");
    expect(payload.assembly_references[0]).toMatchObject({ status: "research", id: "nezha-v2-ultrasonic-gate" });
    expect(payload.assembly_references[0].content_hash).toMatch(/^[a-f0-9]{64}$/);
    expect((await page.request.get("/api/circuit/catalog?kit_id=elecfreaks-ef08288", { headers })).status()).toBe(404);

    await page.setViewportSize({ width: 1440, height: 1080 });
    await page.goto(`/${ws.slug}/circuit`);
    await page.getByRole("link", { name: /超声波感应门.*查看装配参考/ }).click();
    await expect(page).toHaveURL(/\/circuit\?reference=nezha-v2-ultrasonic-gate$/);
    await expect(page.getByRole("heading", { name: "超声波感应门", exact: true })).toBeVisible();
    await expect(page.getByText("图示 16 类 · 36 件 · 15 步装配")).toBeVisible();
    await expect(page.getByRole("button", { name: "上一步参考" })).toBeDisabled();
    await page.screenshot({ path: testInfo.outputPath("assembly-reference-desktop.png"), fullPage: true });
    for (let step = 2; step <= 15; step++) {
      await page.getByRole("button", { name: "下一步参考" }).click();
      await expect(page.getByText(`参考第 ${step} / 15 步`)).toBeVisible();
    }
    await expect(page.getByRole("button", { name: "下一步参考" })).toBeDisabled();
    await expect(page.getByRole("link", { name: "查看这一步的原图" })).toHaveAttribute("href", /step-31-16.png$/);
    await page.getByRole("heading", { name: "图示材料", exact: true }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: testInfo.outputPath("assembly-reference-materials.png"), fullPage: true });
    await page.reload();
    await expect(page.getByText("参考第 1 / 15 步")).toBeVisible();
    await page.setViewportSize({ width: 390, height: 844 });
    await page.getByRole("heading", { name: "超声波感应门", exact: true }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: testInfo.outputPath("assembly-reference-narrow.png"), fullPage: true });
    expect(await page.locator("main.circuit-studio").evaluate((node) => node.scrollWidth <= node.clientWidth)).toBe(true);
    await page.goto(`/${ws.slug}/circuit?reference=unavailable`);
    await expect(page.locator("main.circuit-studio").getByRole("alert")).toContainText("暂时无法读取这件装配参考");
    expect(writes).toEqual([]);
    expect(errors).toEqual([]);
  } finally {
    await api.deleteWorkspace();
    await api.cleanup();
  }
});
