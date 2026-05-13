import { test, expect } from "@playwright/test";

// This test exercises the Ctrl+F find widget inside the diff viewer.
// It verifies that:
//   1. Ctrl+F opens Monaco's find widget (not the browser's)
//   2. Typing in the find widget works (keys like "." don't trigger navigation)
//   3. Escape closes the find widget without closing the diff viewer

test.describe("Diff viewer find widget", () => {
  test("Ctrl+F opens Monaco find, typing works, Escape closes find not viewer", async ({
    page,
    request,
  }) => {
    test.setTimeout(60000);

    // The test server runs inside the shelley repo, which is a git repo.
    // We need the shelley root dir. Fetch it from the git diffs API using
    // the default CWD (the server's working dir).
    const cwdResp = await request.get("/api/git/diffs?cwd=.");
    expect(cwdResp.ok()).toBeTruthy();
    const cwdData = await cwdResp.json();
    const gitRoot = cwdData.gitRoot;
    expect(gitRoot).toBeTruthy();

    // Create a conversation with CWD set to the git root so the diff button appears.
    const newResp = await request.post("/api/conversations/new", {
      data: { message: "Hello", model: "predictable", cwd: gitRoot },
    });
    expect(newResp.ok()).toBeTruthy();
    const { conversation_id } = await newResp.json();

    // Wait for agent reply so the conversation is fully loaded.
    let slug = "";
    await expect(async () => {
      const resp = await request.get(`/api/conversation/${conversation_id}`);
      const body = await resp.json();
      const done = body.messages?.some(
        (m: { type: string; end_of_turn?: boolean }) =>
          m.type === "agent" && m.end_of_turn === true,
      );
      expect(done).toBeTruthy();
      slug = body.conversation?.slug || "";
      expect(slug).toBeTruthy();
    }).toPass({ timeout: 15000 });

    // Navigate to the conversation.
    await page.goto(`/c/${slug}`);
    await page.waitForLoadState("domcontentloaded");

    // Open the overflow menu and click the diffs button.
    const overflowBtn = page.locator(".chat-overflow-menu-wrapper .btn-icon");
    await expect(overflowBtn).toBeVisible({ timeout: 10000 });
    await overflowBtn.click();

    const diffsBtn = page.locator(".overflow-menu-item").filter({ hasText: /diffs/i });
    await expect(diffsBtn).toBeVisible();
    await diffsBtn.click();

    // Wait for the diff viewer overlay to appear.
    const overlay = page.locator(".diff-viewer-overlay");
    await expect(overlay).toBeVisible({ timeout: 10000 });

    // When the diff viewer falls back to the most recent commit (clean tree in
    // CI), it prepends a synthetic "commit-message:" pseudo-file and auto-
    // selects it. That pseudo-file does NOT mount Monaco, so the find widget
    // would never appear. Pick the first real (non-commit-message) file by
    // filtering on the header text — the parent .diff-viewer-file-item also
    // contains the expanded Monaco content, which can spuriously match if the
    // currently-shown file happens to contain the filter string. With a dirty
    // tree (working changes), there are no commit-message entries and the
    // first real file is already expanded — the guard skips the click then.
    const firstRealFileHeader = overlay
      .locator(".diff-viewer-file-item-header")
      .filter({ hasNotText: /commit-message:/ })
      .first();
    await expect(firstRealFileHeader).toBeVisible({ timeout: 10000 });
    const alreadyExpanded = await firstRealFileHeader.evaluate(
      (el) => el.parentElement?.classList.contains("expanded") ?? false,
    );
    if (!alreadyExpanded) {
      await firstRealFileHeader.click();
    }

    // Now wait for the Monaco editor to render inside the expanded row's slot.
    const editorContainer = overlay.locator(".diff-viewer-editor");
    await expect(async () => {
      const visible = await editorContainer.isVisible();
      expect(visible).toBeTruthy();
      const monacoEl = await editorContainer.locator(".monaco-editor").count();
      expect(monacoEl).toBeGreaterThan(0);
    }).toPass({ timeout: 30000 });

    // Verify the find widget is NOT visible initially.
    const findWidget = editorContainer.locator(".find-widget.visible");
    await expect(findWidget).toHaveCount(0);

    // Press Ctrl+F to open the find widget.
    await page.keyboard.press("Control+f");

    // Wait for the find widget to become visible.
    await expect(findWidget).toBeVisible({ timeout: 5000 });

    // Set the find query directly. We use fill() rather than typing each
    // character because under touch viewports Monaco's find input loses focus
    // between keystrokes intermittently, which makes typing flaky here. We
    // separately verify below that the "." navigation shortcut doesn't fire
    // when the find widget is open.
    const findInput = findWidget.getByRole("textbox", { name: "Find" });
    await findInput.fill("test.file");
    await expect(findInput).toHaveValue(/test\.file/, { timeout: 5000 });

    // With the find widget open and focused, pressing "." must NOT trigger
    // the diff viewer's "next change" navigation shortcut. If the shortcut
    // fired it would steal focus from the find widget; assert focus stays.
    await findInput.press(".");
    await expect(findInput).toBeFocused();

    // The diff viewer should still be open.
    await expect(overlay).toBeVisible();

    // Press Escape to close the find widget.
    await page.keyboard.press("Escape");

    // The find widget should be hidden now.
    await expect(findWidget).toHaveCount(0, { timeout: 5000 });

    // The diff viewer should still be open (Escape only closed the find widget).
    await expect(overlay).toBeVisible();

    // Now press Escape again to close the diff viewer.
    await page.keyboard.press("Escape");
    await expect(overlay).toHaveCount(0, { timeout: 5000 });
  });
});
