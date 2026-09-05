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
    await page.context().addCookies(
      api.getAuthCookies().map((cookie) => ({
        ...cookie,
        url: baseURL!,
        httpOnly: cookie.name === "chimii_auth",
        sameSite: "Lax" as const,
      })),
    );
    await page.addInitScript(() => {
      localStorage.setItem("chimii:chat:isOpen", "false");
      if (!localStorage.getItem("chimii_locale"))
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
    await expect(
      page.getByRole("button", { name: "Start this project" }),
    ).toBeDisabled();
    await page.getByRole("button", { name: /I have this kit/ }).click();
    await page.getByRole("spinbutton", { name: /^FM/ }).fill("0");
    await page.getByRole("checkbox", { name: /I checked the model/ }).check();
    await page.getByRole("button", { name: "Save parts box" }).click();
    await expect(page.getByText("Parts box saved.")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Start this project" }),
    ).toBeDisabled();
    await page.getByRole("button", { name: "Edit saved quantities" }).click();
    await page.getByRole("spinbutton", { name: /^FM/ }).fill("1");
    await page.getByRole("checkbox", { name: /I checked the model/ }).check();
    await page.getByRole("button", { name: "Save parts box" }).click();
    await expect(
      page.getByRole("button", { name: "Start this project" }),
    ).toBeEnabled();
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

test("saves a BOSON box, follows keyed connections and records a separate family trial", async ({
  page,
  baseURL,
}, testInfo) => {
  test.setTimeout(150_000);
  const api = new TestApiClient();
  await api.login(`boson-${Date.now()}@example.test`, "BOSON browser test");
  const workspace = await api.ensureWorkspace(
    "BOSON browser test",
    `boson-${Date.now()}`,
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
    await page.addInitScript(() => {
      localStorage.setItem("chimii:chat:isOpen", "false");
      if (!localStorage.getItem("chimii_locale"))
        localStorage.setItem("chimii_locale", "en");
    });
    await page.setViewportSize({ width: 1440, height: 1080 });
    await page.goto(`/${workspace.slug}/circuit`, {
      waitUntil: "domcontentloaded",
    });
    await page.locator("#circuit-kit").selectOption("dfrobot-edu0080-en");
    await expect(
      page.getByRole("button", { name: "Start this project" }),
    ).toBeDisabled();
    await page.getByRole("button", { name: /I have this kit/ }).click();
    await page.getByRole("checkbox", { name: /I checked the model/ }).check();
    await page.getByRole("button", { name: "Save parts box" }).click();
    await expect(page.getByText("Parts box saved.")).toBeVisible();
    await page.reload();
    await page.locator("#circuit-kit").selectOption("dfrobot-edu0080-en");
    await expect(
      page.getByRole("button", { name: "Start this project" }),
    ).toBeEnabled();
    await expect(page.getByText("Parts checked · revision 1")).toBeVisible();
    await page.getByRole("button", { name: /Two clues for a fan/ }).click();
    await expect(
      page.getByRole("button", { name: /Two clues for a fan/ }),
    ).toHaveAttribute("aria-pressed", "true");
    await page.screenshot({
      path: testInfo.outputPath("boson-home.png"),
      fullPage: true,
    });
    await page.getByRole("button", { name: "Start this project" }).click();
    await expect(page.getByText("STEP 1 / 10")).toBeVisible();
    const id = new URL(page.url()).pathname.split("/").at(-1)!;
    await page.getByRole("button", { name: "Complete layout" }).click();
    await expect(page.locator("[data-connection-id]")).toHaveCount(5);
    await expect(
      page.getByRole("combobox", { name: "Highlight a layer" }),
    ).toHaveCount(0);
    await page.screenshot({
      path: testInfo.outputPath("boson-connections.png"),
      fullPage: true,
    });
    for (let step = 1; step < 10; step++) {
      await page.getByRole("button", { name: "Done, next step" }).click();
      await expect(page.getByText(`Saved at step ${step + 1}`)).toBeVisible();
    }
    await page.getByText("Physical build notes", { exact: true }).click();
    await page
      .getByRole("textbox", { name: "Model and markings on your actual kit" })
      .fill("EDU0080-EN automated test fixture (no real hardware)");
    await page
      .getByRole("combobox", { name: "Observed result" })
      .selectOption("needs_help");
    await page
      .getByRole("textbox", { name: /What happened/ })
      .fill("Automated persistence test only; no physical trial took place.");
    await page
      .getByRole("checkbox", { name: /An adult checked the assembly/ })
      .check();
    await page.getByRole("button", { name: "Save this trial" }).click();
    await expect(
      page.getByText("Trial saved as a family report."),
    ).toBeVisible();
    await page.reload();
    await expect(page.getByText("STEP 10 / 10")).toBeVisible();
    await page.getByText("Physical build notes", { exact: true }).click();
    await expect(page.getByText("1 recent trials (up to 50)")).toBeVisible();
    await expect(
      page.getByText(
        "Automated persistence test only; no physical trial took place.",
      ),
    ).toBeVisible();
    const doc = await page.request.get(`/api/circuit/creations/${id}`, {
      headers: { "X-Workspace-ID": workspace.id },
    });
    const stored = await doc.json();
    expect(stored.document.version).toBe(2);
    expect(stored.document.inventory_revision).toBe(1);
    expect(stored.document.validation.physical_verification).toBe("not_tested");
    const download = page.waitForEvent("download");
    await page
      .getByRole("button", { name: "Export design and trial notes" })
      .click();
    expect((await download).suggestedFilename()).toContain("trials.json");
    await page.setViewportSize({ width: 390, height: 844 });
    await page.getByRole("button", { name: "Complete layout" }).click();
    await page.screenshot({
      path: testInfo.outputPath("boson-mobile.png"),
      fullPage: true,
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    const setLanguage = async (language: string) => {
      const token = (await page.context().cookies()).find(
        (c) => c.name === "chimii_csrf",
      )!.value;
      const result = await page.request.patch("/api/me", {
        headers: { "X-CSRF-Token": token, "X-Workspace-ID": workspace.id },
        data: { language },
      });
      expect(result.ok()).toBe(true);
      await page
        .context()
        .addCookies([
          {
            name: "chimii-locale",
            value: language,
            url: baseURL!,
            sameSite: "Lax",
          },
        ]);
    };
    await setLanguage("zh-Hans");
    await page.reload();
    await expect(page.getByText("实物试搭记录", { exact: true })).toBeVisible();
    await page.setViewportSize({ width: 1440, height: 1080 });
    await page.getByRole("button", { name: "完整搭建图" }).click();
    await page.screenshot({
      path: testInfo.outputPath("boson-chinese.png"),
      fullPage: true,
    });
    await setLanguage("en");
    await page.reload();
    const csrf = (await page.context().cookies()).find(
      (c) => c.name === "chimii_csrf",
    )!.value;
    const headers = { "X-CSRF-Token": csrf, "X-Workspace-ID": workspace.id };
    const profile = await page.request.post("/api/child-profiles", {
      headers,
      data: { display_name: "Module builder", pin: "2468" },
    });
    expect(profile.status()).toBe(201);
    const entered = await page.request.post(
      `/api/child-profiles/${(await profile.json()).id}/enter`,
      { headers },
    );
    expect(entered.ok()).toBe(true);
    await page.goto(`/${workspace.slug}/circuit`, {
      waitUntil: "domcontentloaded",
    });
    await page.locator("#circuit-kit").selectOption("dfrobot-edu0080-en");
    await expect(
      page.getByRole("button", { name: "Start this project" }),
    ).toBeEnabled();
    await expect(
      page.getByRole("button", { name: "Edit saved quantities" }),
    ).toHaveCount(0);
    const denied = await page.request.put(
      "/api/circuit/inventory/dfrobot-edu0080-en",
      { headers, data: {} },
    );
    expect(denied.status()).toBe(403);
    const privateTrials = await page.request.get(
      `/api/circuit/creations/${id}/trials`,
      { headers: { "X-Workspace-ID": workspace.id } },
    );
    expect(privateTrials.status()).toBe(404);
    await page.getByRole("button", { name: "Start this project" }).click();
    await expect(page.getByText("STEP 1 / 6")).toBeVisible();
    expect(errors).toEqual([]);
  } finally {
    await api.deleteWorkspace();
  }
});
