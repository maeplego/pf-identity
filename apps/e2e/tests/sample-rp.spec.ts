import { expect, test } from "@playwright/test";

const idp = "http://localhost:18080";
const rp = "http://localhost:13001";

test("sample-rp login shows UserInfo then logout", async ({ page }) => {
  const email = `e2e-${Date.now()}@example.com`;
  await page.goto(rp);
  await page.getByTestId("login-link").click();
  await page.getByRole("link", { name: "登録へ" }).click();
  await page.getByLabel("メール").fill(email);
  await page.getByLabel("表示名").fill("E2E");
  await page.getByLabel("パスワード").fill("long-enough");
  await page.getByRole("button", { name: "登録" }).click();
  await page.getByRole("button", { name: "許可" }).click();
  await expect(page.getByTestId("userinfo")).toContainText(email);
  await page.getByTestId("logout").click();
  await expect(page.getByTestId("login-link")).toBeVisible();
});

test("altered redirect_uri is rejected at the IdP", async ({ page }) => {
  const q = new URLSearchParams({
    response_type: "code",
    client_id: "sample-rp",
    redirect_uri: "http://localhost:13001/callback?extra=1",
    scope: "openid",
    code_challenge: "a".repeat(43),
    code_challenge_method: "S256",
  });
  const res = await page.goto(`${idp}/authorize?${q.toString()}`);
  expect(res?.status()).toBe(400);
});
