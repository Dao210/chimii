import { expect, test } from "@playwright/test";
import { TestApiClient } from "./fixtures";

test("composes a new digital circuit, explores signals and preserves conditions on revision", async ({
  page,
  baseURL,
}, testInfo) => {
  test.setTimeout(180_000);
  const api = new TestApiClient();
  await api.login(
    `composition-${Date.now()}@example.test`,
    "Composition acceptance",
  );
  const ws = await api.ensureWorkspace(
    "Composition acceptance",
    `composition-${Date.now()}`,
  );
  await api.markUserOnboarded();
  try {
    await page.context().addCookies(
      api.getAuthCookies().map((cookie) => ({
        ...cookie,
        url: baseURL!,
        sameSite: "Lax" as const,
        httpOnly: cookie.name === "chimii_auth",
      })),
    );
    await page
      .context()
      .addCookies([{ name: "chimii-locale", value: "en", url: baseURL! }]);
    await page.addInitScript(() =>
      localStorage.setItem("chimii:chat:isOpen", "false"),
    );
    await page.setViewportSize({ width: 1440, height: 1080 });
    await page.goto(`/${ws.slug}/circuit`);
    await page
      .getByLabel("My supported kit")
      .selectOption("dfrobot-edu0080-en");
    await page.getByRole("button", { name: /I have this kit/ }).click();
    await page.getByRole("checkbox", { name: /I checked the model/ }).check();
    await page.getByRole("button", { name: "Save parts box" }).click();
    await expect(page.getByText(/My parts box · Parts checked/)).toBeVisible();
    await page
      .getByRole("button", { name: "A fan requiring both button and motion" })
      .click();
    await page
      .getByRole("region", { name: "Discuss this circuit" })
      .getByRole("button", { name: "Send", exact: true })
      .click();
    await expect(page).toHaveURL(/\/circuit\?conversation=/);
    const preview = page.getByRole("region", { name: "Try the signal states" });
    await expect(preview).toBeVisible({ timeout: 30000 });
    await expect(preview.getByText("5 logic cases checked")).toBeVisible();
    const conversationId = new URL(page.url()).searchParams.get(
      "conversation",
    )!;
    const csrf = (await page.context().cookies()).find(
      (c) => c.name === "chimii_csrf",
    )!.value;
    const headers = { "X-Workspace-ID": ws.id, "X-CSRF-Token": csrf };
    const readConversation = async () => {
      const res = await page.request.get(
        `/api/build/conversations/${conversationId}`,
        { headers },
      );
      expect(res.ok()).toBe(true);
      return res.json();
    };
    const first = await readConversation();
    const id = first.session.circuit_creation_id;
    const readCreation = async (creationId: string) => {
      const res = await page.request.get(
        `/api/circuit/creations/${creationId}`,
        { headers },
      );
      expect(res.ok()).toBe(true);
      return res.json();
    };
    const original = await readCreation(id);
    expect(original.document.composition).toEqual({
      inputs: ["BOS0002-R", "BOS0013"],
      operation: "and",
      output: "BOS0021",
    });
    expect(original.document.validation.physical_verification).toBe(
      "not_tested",
    );
    await expect(preview.getByRole("status")).toHaveText(/Off/);
    await preview.getByRole("button", { name: /i2r/ }).click();
    await expect(preview.getByRole("status")).toHaveText(/Off/);
    await preview.getByRole("button", { name: /i13/ }).click();
    await expect(preview.getByRole("status")).toHaveText(/On/);
    await preview.getByRole("button", { name: /Preview power/ }).click();
    await expect(preview.getByRole("status")).toHaveText(/Off/);
    expect(await readCreation(id)).toEqual(original);
    await preview.getByRole("button", { name: /Preview power/ }).click();
    await expect(preview.getByRole("status")).toHaveText(/On/);
    await page.getByRole("button", { name: "Done, next step" }).click();
    await expect(page.getByTestId("connection-focus")).toBeVisible();
    await expect(
      page.getByTestId("connection-focus").locator("[data-module-id]"),
    ).toHaveCount(2);
    await page.locator("main.circuit-studio").evaluate((el) => {
      el.scrollTop = 0;
    });
    await page.screenshot({
      path: testInfo.outputPath("composition-desktop.png"),
      fullPage: true,
    });
    await preview.screenshot({
      path: testInfo.outputPath("composition-logic.png"),
    });
    await page
      .getByTestId("connection-focus")
      .screenshot({ path: testInfo.outputPath("composition-connection.png") });
    await page
      .getByRole("button", { name: "Complete layout", exact: true })
      .click();
    await page
      .getByTestId("circuit-module-board")
      .first()
      .screenshot({ path: testInfo.outputPath("composition-modules.png") });
    await page
      .getByRole("button", { name: "Follow steps", exact: true })
      .click();
    await page
      .getByRole("textbox", { name: "Discuss this circuit" })
      .fill("change to a light, keep both conditions");
    await page
      .getByRole("region", { name: "Discuss this circuit" })
      .getByRole("button", { name: "Send", exact: true })
      .click();
    await expect
      .poll(async () => (await readConversation()).messages.length, {
        timeout: 30000,
      })
      .toBe(4);
    const second = await readConversation();
    const revised = await readCreation(second.session.circuit_creation_id);
    expect(revised.document.composition).toEqual({
      ...original.document.composition,
      output: "BOS0017-R",
    });
    expect(revised.document.content_hash).not.toBe(
      original.document.content_hash,
    );
    expect((await readCreation(id)).document).toEqual(original.document);
    await page.reload();
    await expect(preview).toBeVisible();
    await page.getByRole("button", { name: "Done, next step" }).click();
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(page.getByTestId("connection-focus")).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.getByTestId("connection-focus").scrollIntoViewIfNeeded();
    await page.screenshot({
      path: testInfo.outputPath("composition-mobile.png"),
      fullPage: true,
    });
    const localeRes = await page.request.patch("/api/me", {
      headers,
      data: { language: "zh-Hans" },
    });
    expect(localeRes.ok()).toBe(true);
    await page
      .context()
      .addCookies([{ name: "chimii-locale", value: "zh-Hans", url: baseURL! }]);
    await page.reload();
    await expect(
      page.getByRole("region", { name: "试一试信号变化" }),
    ).toBeVisible();
    await page.getByRole("region", { name: "试一试信号变化" }).screenshot({
      path: testInfo.outputPath("composition-chinese-logic.png"),
    });
    await page.getByTestId("connection-focus").screenshot({
      path: testInfo.outputPath("composition-chinese-connection.png"),
    });
    await page.screenshot({
      path: testInfo.outputPath("composition-chinese-mobile.png"),
      fullPage: true,
    });
  } finally {
    await api.cleanup();
  }
});
