import { expect, test } from "@playwright/test";
import { TestApiClient } from "./fixtures";

test("builds and resumes a radio from real saved circuit data", async ({
  page,
  baseURL,
}, testInfo) => {
  test.setTimeout(120_000);
  const api = new TestApiClient();
  await api.login(`circuit-${Date.now()}@example.test`, "Circuit browser test");
  const workspace = await api.ensureWorkspace(
    "Circuit browser test",
    `circuit-${Date.now()}`,
  );
  await api.markUserOnboarded();
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  try {
    await page
      .context()
      .addCookies(
        api
          .getAuthCookies()
          .map((cookie) => ({
            ...cookie,
            url: baseURL!,
            httpOnly: cookie.name === "chimii_auth",
            sameSite: "Lax" as const,
          })),
      );
    await page.addInitScript(() => {
      localStorage.setItem("chimii:chat:isOpen", "false");
      localStorage.setItem("chimii_locale", "en");
    });
    await page.setViewportSize({ width: 1440, height: 1080 });
    await page.goto(`/${workspace.slug}/circuit`, {
      waitUntil: "domcontentloaded",
    });
    await expect(
      page.getByRole("heading", { name: "Build a little wonder" }),
    ).toBeVisible({ timeout: 20_000 });
    await page.getByRole("button", { name: /My tabletop radio/ }).click();
    await page.getByText("Check or change quantities").click();
    await page.getByRole("spinbutton", { name: /^FM/ }).fill("0");
    await expect(
      page.getByRole("button", { name: "Start this project" }),
    ).toBeDisabled();
    await page.getByRole("spinbutton", { name: /^FM/ }).fill("1");
    await page.screenshot({
      path: testInfo.outputPath("circuit-home.png"),
      fullPage: true,
    });
    await page.getByRole("button", { name: "Start this project" }).click();
    await expect(page).toHaveURL(
      new RegExp(`/${workspace.slug}/circuit/[^/]+$`),
    );
    const parentCreationID = new URL(page.url()).pathname.split("/").at(-1)!;
    await expect(page.getByText("STEP 1 / 23")).toBeVisible();
    await page.getByRole("button", { name: "Done, next step" }).click();
    await expect(page.getByText("Saved at step 2")).toBeVisible();
    await page.reload();
    await expect(page.getByText("STEP 2 / 23")).toBeVisible();
    await page.getByRole("button", { name: "Complete layout" }).click();
    await expect(page.locator("[data-placement]")).toHaveCount(23);
    await page.screenshot({
      path: testInfo.outputPath("circuit-radio.png"),
      fullPage: true,
    });
    const download = page.waitForEvent("download");
    await page.getByRole("button", { name: "Download design" }).click();
    expect((await download).suggestedFilename()).toMatch(/\.json$/);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.getByRole("button", { name: "Enlarge diagram" }).click();
    await expect(
      page.getByRole("button", { name: "Fit diagram" }),
    ).toBeVisible();
    expect(
      await page
        .getByRole("region", { name: "My tabletop radio" })
        .evaluate((element) => element.scrollWidth > element.clientWidth),
    ).toBe(true);
    await page.screenshot({
      path: testInfo.outputPath("circuit-mobile.png"),
      fullPage: true,
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);

    const csrf = (await page.context().cookies()).find(
      (cookie) => cookie.name === "chimii_csrf",
    )!.value;
    const headers = { "X-CSRF-Token": csrf, "X-Workspace-ID": workspace.id };
    const profile = await page.request.post("/api/child-profiles", {
      headers,
      data: { display_name: "Little builder", pin: "2468" },
    });
    expect(profile.status()).toBe(201);
    const profileID = (await profile.json()).id;
    const entered = await page.request.post(
      `/api/child-profiles/${profileID}/enter`,
      { headers },
    );
    expect(entered.ok()).toBe(true);
    await page.goto(`/${workspace.slug}/circuit`, {
      waitUntil: "domcontentloaded",
    });
    await expect(
      page.getByRole("heading", { name: "Build a little wonder" }),
    ).toBeVisible({ timeout: 20_000 });
    const denied = await page.request.get(
      `/api/circuit/creations/${parentCreationID}`,
      { headers: { "X-Workspace-ID": workspace.id } },
    );
    expect(denied.status()).toBe(404);
    await page.getByRole("button", { name: "Start this project" }).click();
    await expect(page.getByText("STEP 1 / 8")).toBeVisible();
    await expect(page).toHaveURL(
      new RegExp(`/${workspace.slug}/circuit/[^/]+$`),
    );
    expect(errors).toEqual([]);
  } finally {
    await api.deleteWorkspace();
  }
});
