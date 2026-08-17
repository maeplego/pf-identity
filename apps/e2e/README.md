# Browser e2e

DESIGN のデモ経路を Playwright で固定します。**本番の品質ゲートではありません。**

同じ認可コードの二回交換はブラウザより HTTP テストの方が正確なので、`apps/server` の `TestAuthorizationCodePKCEAndReplay` が正本です。

```powershell
cd apps/e2e
npm install
npx playwright install chromium
$env:GOTOOLCHAIN = "local"
npm test
```

IdP / sample-rp / sample-rp-b / admin はテストが専用ポート（18080 / 13001 / 13003 / 13002）で起動します。

- `backchannel-logout.spec.ts` — `/backchannel-logout` が不正な `logout_token` を拒否する
- `cross-rp-logout.spec.ts` — 2 つの RP でログイン後、片方から logout するともう片方のセッションも終わる
