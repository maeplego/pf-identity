# admin

P01 のオペレーター UI です。**本番の管理コンソールではありません。**

ブラウザは IdP の admin トークンを持ちません。Next.js が `IDENTITY_ADMIN_TOKEN` を使って `/admin/api/*` を呼びます。

## 起動

IdP に `IDENTITY_ADMIN_TOKEN` を設定してから:

```powershell
copy .env.example .env.local
npm install
npm run dev
```

http://localhost:3002
