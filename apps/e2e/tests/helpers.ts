import { expect, type APIRequestContext, type Page } from "@playwright/test";

export const idp = "http://localhost:18080";
export const rpA = "http://localhost:13001";
export const rpB = "http://localhost:13003";
export const adminToken = "e2e-admin-token";

export async function ensurePublicClient(
  request: APIRequestContext,
  opts: { id: string; name: string; redirectUri: string },
) {
  const base = opts.redirectUri.replace(/\/callback$/, "");
  const res = await request.post(`${idp}/admin/api/clients`, {
    headers: { Authorization: `Bearer ${adminToken}` },
    data: {
      id: opts.id,
      name: opts.name,
      type: "public",
      redirect_uris: [opts.redirectUri],
      post_logout_redirect_uris: [`${base}/logged-out`],
      frontchannel_logout_uri: `${base}/frontchannel-logout`,
      backchannel_logout_uri: `${base}/backchannel-logout`,
    },
  });
  if (res.status() === 409) {
    return;
  }
  expect(res.ok(), `ensurePublicClient ${opts.id}: ${res.status()} ${await res.text()}`).toBeTruthy();
}

export async function registerAndLogin(page: Page, rpUrl: string, email: string) {
  await page.goto(rpUrl);
  await page.getByTestId("login-link").click();
  await page.getByRole("link", { name: "登録へ" }).click();
  await page.getByLabel("メール").fill(email);
  await page.getByLabel("表示名").fill("E2E");
  await page.getByLabel("パスワード").fill("long-enough");
  await page.getByRole("button", { name: "登録" }).click();
  await page.getByRole("button", { name: "許可" }).click();
  await expect(page.getByTestId("userinfo")).toContainText(email);
}

export async function loginViaIdp(page: Page, rpUrl: string) {
  await page.goto(rpUrl);
  await page.getByTestId("login-link").click();
  await page.getByRole("button", { name: "許可" }).click();
  await expect(page.getByTestId("userinfo")).toBeVisible();
}
