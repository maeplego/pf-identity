import { expect, test } from "@playwright/test";

import { ensurePublicClient, loginViaIdp, registerAndLogin, rpA, rpB } from "./helpers";

test.beforeAll(async ({ request }) => {
  await ensurePublicClient(request, {
    id: "sample-rp-b",
    name: "Sample RP B",
    redirectUri: `${rpB}/callback`,
  });
});

test("logging out from one RP ends the other RP session", async ({ browser }) => {
  const email = `cross-rp-${Date.now()}@example.com`;
  const context = await browser.newContext();
  const pageA = await context.newPage();
  const pageB = await context.newPage();

  await registerAndLogin(pageA, rpA, email);
  await expect(pageA.getByTestId("rp-title")).toHaveText("sample-rp");

  await loginViaIdp(pageB, rpB);
  await expect(pageB.getByTestId("rp-title")).toHaveText("sample-rp-b");
  await expect(pageB.getByTestId("userinfo")).toContainText(email);

  await pageA.getByTestId("logout").click();
  await expect(pageA.getByTestId("login-link")).toBeVisible();

  await pageB.reload();
  await expect(pageB.getByTestId("login-link")).toBeVisible();
  await expect(pageB.getByTestId("userinfo")).toHaveCount(0);

  await context.close();
});
