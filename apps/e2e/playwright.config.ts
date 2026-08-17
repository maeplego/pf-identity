import { defineConfig, devices } from "@playwright/test";
import path from "node:path";

const idpPort = 18080;
const rpPort = 13001;
const adminPort = 13002;
const apps = path.join(__dirname, "..");
const idp = `http://localhost:${idpPort}`;
const rp = `http://localhost:${rpPort}`;
const admin = `http://localhost:${adminPort}`;

export default defineConfig({
  testDir: "./tests",
  timeout: 120_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: "list",
  use: {
    ...devices["Desktop Chrome"],
    trace: "off",
  },
  webServer: [
    {
      command: "go run ./cmd/server",
      cwd: path.join(apps, "server"),
      url: `${idp}/health`,
      reuseExistingServer: false,
      timeout: 120_000,
      env: {
        ...process.env,
        GOTOOLCHAIN: "local",
        IDENTITY_HTTP_ADDR: `:${idpPort}`,
        IDENTITY_ISSUER: idp,
        IDENTITY_DEV_GENERATE_KEYS: "true",
        IDENTITY_STORE: "memory",
        IDENTITY_COOKIE_SECURE: "false",
        IDENTITY_ADMIN_TOKEN: "e2e-admin-token",
        IDENTITY_SEED_PUBLIC_CLIENT_ID: "sample-rp",
        IDENTITY_SEED_PUBLIC_REDIRECT_URI: `${rp}/callback`,
      },
    },
    {
      command: "npx next dev -p 13001 --hostname localhost",
      cwd: path.join(apps, "sample-rp"),
      url: rp,
      reuseExistingServer: false,
      timeout: 120_000,
      env: {
        ...process.env,
        OIDC_ISSUER: idp,
        OIDC_CLIENT_ID: "sample-rp",
        OIDC_REDIRECT_URI: `${rp}/callback`,
        OIDC_POST_LOGOUT_REDIRECT_URI: `${rp}/logged-out`,
      },
    },
    {
      command: "npx next dev -p 13002 --hostname localhost",
      cwd: path.join(apps, "admin"),
      url: admin,
      reuseExistingServer: false,
      timeout: 120_000,
      env: {
        ...process.env,
        IDENTITY_API_BASE: idp,
        IDENTITY_ADMIN_TOKEN: "e2e-admin-token",
      },
    },
  ],
});
