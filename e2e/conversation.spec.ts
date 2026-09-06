import { expect, test } from "@playwright/test";
import { TestApiClient } from "./fixtures";

test("one saved conversation produces independent circuit versions and a shared gallery", async ({
  page,
  baseURL,
}, testInfo) => {
  test.setTimeout(180_000);
  const api = new TestApiClient();
  await api.login(
    `conversation-${Date.now()}@example.test`,
    "Conversation acceptance",
  );
  const ws = await api.ensureWorkspace(
    "Conversation acceptance",
    `conversation-${Date.now()}`,
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
    await expect(
      page.getByRole("heading", { name: "Creation studio" }),
    ).toBeVisible();
    await page.getByRole("button", { name: /I have this kit/ }).click();
    await page.getByRole("checkbox", { name: /I checked the model/ }).check();
    await page.getByRole("button", { name: "Save parts box" }).click();
    await expect(page.getByText("Parts box saved.")).toBeVisible();
    await page
      .getByRole("button", { name: "My tabletop radio", exact: true })
      .click();
    await expect(page).toHaveURL(/\/circuit\?conversation=/);
    await expect(page.getByText("STEP 1 / 23")).toBeVisible({ timeout: 30000 });
    const conversationID = new URL(page.url()).searchParams.get(
      "conversation",
    )!;
    const csrf = (await page.context().cookies()).find(
      (c) => c.name === "chimii_csrf",
    )!.value;
    const headers = { "X-Workspace-ID": ws.id, "X-CSRF-Token": csrf };
    const readConversation = async () => {
      const res = await page.request.get(
        `/api/build/conversations/${conversationID}`,
        { headers },
      );
      expect(res.ok()).toBe(true);
      return res.json();
    };
    const first = await readConversation();
    expect(first.messages).toHaveLength(2);
    expect(first.session.kind).toBe("circuit");
    const id = first.session.circuit_creation_id;
    const original = await (
      await page.request.get(`/api/circuit/creations/${id}`, { headers })
    ).json();
    await page.getByRole("button", { name: "Done, next step" }).click();
    await expect(page.getByText("Saved at step 2")).toBeVisible();
    await page.reload();
    await expect(page.getByText("STEP 2 / 23")).toBeVisible();
    await page.getByRole("button", { name: "Complete layout" }).click();
    await expect(page.locator("[data-placement]")).toHaveCount(23);
    const download = page.waitForEvent("download");
    await page.getByRole("button", { name: "Download design" }).click();
    expect((await download).suggestedFilename()).toMatch(/\.json$/);
    await page.screenshot({
      path: testInfo.outputPath("conversation-radio-desktop.png"),
      fullPage: true,
    });
    // The exact catalogue label is read from the server, avoiding translated fixture drift.
    const catalog = await (
      await page.request.get("/api/circuit/catalog", { headers })
    ).json();
    const lamp = catalog.catalog.projects.find(
      (p: { id: string }) => p.id === "switch-light",
    );
    await page
      .getByRole("button", { name: lamp.title.en, exact: true })
      .click();
    await expect
      .poll(async () => (await readConversation()).messages.length, {
        timeout: 30000,
      })
      .toBe(4);
    const second = await readConversation();
    expect(second.session.id).not.toBe(first.session.id);
    expect(second.session.circuit_creation_id).not.toBe(id);
    const untouched = await (
      await page.request.get(`/api/circuit/creations/${id}`, { headers })
    ).json();
    expect(untouched.document).toEqual(original.document);
    expect(untouched.current_step).toBe(1);
    await expect(
      page.getByRole("heading", { name: lamp.title.en, exact: true }),
    ).toBeVisible();
    await page.reload();
    await expect(
      page.getByRole("heading", { name: lamp.title.en, exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Open this version" }),
    ).toHaveCount(2);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.getByRole("button", { name: "Creation", exact: true }).click();
    await expect(
      page.getByRole("heading", { name: lamp.title.en, exact: true }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: testInfo.outputPath("conversation-mobile.png"),
      fullPage: true,
    });
    // Chinese conversation and result survive refresh on a narrow display.
    expect(
      (
        await page.request.patch("/api/me", {
          headers,
          data: { language: "zh-Hans" },
        })
      ).ok(),
    ).toBe(true);
    await page
      .context()
      .addCookies([{ name: "chimii-locale", value: "zh-Hans", url: baseURL! }]);
    await page.reload();
    await expect(
      page.getByRole("heading", { name: "创作工作室" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "作品", exact: true }).click();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: testInfo.outputPath("conversation-chinese-mobile.png"),
      fullPage: true,
    });
    await page.setViewportSize({ width: 1440, height: 1080 });
    await page.screenshot({
      path: testInfo.outputPath("conversation-chinese-desktop.png"),
      fullPage: true,
    });
    // Old detail routes and source continuation keep their independent data.
    await page.goto(`/${ws.slug}/circuit/${id}`);
    await expect(page.getByText("第 2 步 / 共 23 步")).toBeVisible();
    await page.getByRole("link", { name: "继续讨论" }).click();
    await expect(page).toHaveURL(/source_kind=circuit/);
    // A deterministic brick edit uses the existing no-model API and the same queue.
    const brickResponse = await page.request.post("/api/build/sessions", {
      headers,
      data: {
        prompt: "Small blue base",
        client_request_id: crypto.randomUUID(),
        kind: "brick",
        design: {
          version: 1,
          mode: "static",
          shapes: [
            {
              id: "base",
              label: "Base",
              kind: "box",
              operation: "add",
              position: { x: 0, y: 0, z: 0 },
              size: { x: 4, y: 6, z: 2 },
              color: 1,
            },
          ],
        },
      },
    });
    expect(brickResponse.status()).toBe(202);
    const brickRun = await brickResponse.json();
    await expect
      .poll(
        async () => {
          const res = await page.request.get(
            `/api/build/sessions/${brickRun.id}`,
            { headers },
          );
          return (await res.json()).status;
        },
        { timeout: 30000 },
      )
      .toBe("completed");
    const creations = await (
      await page.request.get("/api/build/inventions", { headers })
    ).json();
    expect(
      creations.creations.map((item: { kind: string }) => item.kind).sort(),
    ).toEqual(["brick", "circuit", "circuit"]);
    await page.goto(`/${ws.slug}/build?conversation=${brickRun.id}`);
    await expect(
      page.getByRole("heading", { name: "Small blue base" }),
    ).toBeVisible({ timeout: 30000 });
    await page.goto(`/${ws.slug}/creations`);
    await expect(
      page.getByRole("heading", { name: "Small blue base" }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: lamp.title.en, exact: true }),
    ).toBeVisible();
    await page.screenshot({
      path: testInfo.outputPath("conversation-shared-gallery.png"),
      fullPage: true,
    });
    // Child capabilities allow creation, but never reading the parent's conversation.
    const profile = await page.request.post("/api/child-profiles", {
      headers,
      data: { display_name: "Little inventor", pin: "2468" },
    });
    expect(profile.status()).toBe(201);
    const entered = await page.request.post(
      `/api/child-profiles/${(await profile.json()).id}/enter`,
      { headers },
    );
    expect(entered.ok()).toBe(true);
    expect(
      (
        await page.request.get(`/api/build/conversations/${conversationID}`, {
          headers: { "X-Workspace-ID": ws.id },
        })
      ).status(),
    ).toBe(404);
    expect(
      (
        await page.request.put(
          `/api/circuit/inventory/${catalog.catalog.kit_id}`,
          { headers, data: {} },
        )
      ).status(),
    ).toBe(403);
    await page.goto(`/${ws.slug}/circuit`);
    await expect(
      page.getByRole("heading", { name: "创作工作室" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "修改已保存数量" }),
    ).toHaveCount(0);
    await page
      .getByRole("button", { name: lamp.title.zh, exact: true })
      .click();
    await expect(page).toHaveURL(/conversation=/);
    await expect(
      page.getByRole("heading", { name: lamp.title.zh, exact: true }),
    ).toBeVisible({ timeout: 30000 });
    expect(errors).toEqual([]);
  } finally {
    await api.deleteWorkspace();
  }
});
