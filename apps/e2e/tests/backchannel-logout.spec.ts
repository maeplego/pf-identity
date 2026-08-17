import { expect, test } from "@playwright/test";

import { rpA } from "./helpers";

test("backchannel-logout rejects missing logout_token", async ({ request }) => {
  const res = await request.post(`${rpA}/backchannel-logout`);
  expect(res.status()).toBe(400);
  expect(await res.text()).toContain("missing logout_token");
});

test("backchannel-logout rejects invalid logout_token", async ({ request }) => {
  const res = await request.post(`${rpA}/backchannel-logout`, {
    form: { logout_token: "not-a-jwt" },
  });
  expect(res.status()).toBe(400);
  expect(await res.text()).toContain("invalid logout_token");
});
