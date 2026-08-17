import { expect, test } from "@playwright/test";

const admin = "http://localhost:13002";

test("admin can register a public client and open audits", async ({ page }) => {
  const clientId = `e2e-${Date.now()}`;
  await page.goto(`${admin}/clients/new`);
  await page.locator('input[name="id"]').fill(clientId);
  await page.getByLabel("name").fill("E2E RP");
  await page.getByLabel("type").selectOption("public");
  await page.locator('textarea[name="redirect_uris"]').fill("http://localhost:13001/callback");
  await page.getByRole("button", { name: "作成" }).click();
  await expect(page.getByRole("heading", { name: clientId })).toBeVisible();
  await page.goto(`${admin}/audits`);
  await expect(page.getByRole("heading", { name: "監査ログ" })).toBeVisible();
});
