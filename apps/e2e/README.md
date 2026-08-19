# ブラウザ e2e

ログイン、redirect 拒否、2 つの RP のログアウト連動を Playwright で固定します。品質ゲートの本線ではありません。同じ認可コードの二回交換は `apps/server` の `go test` の方が正確です。

```powershell
cd apps/e2e
npm install
npx playwright install chromium
$env:GOTOOLCHAIN = "local"
npm test
```

テストが IdP / sample-rp / sample-rp-b / admin を専用ポート（18080 / 13001 / 13003 / 13002）で起動します。
