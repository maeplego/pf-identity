# 管理 UI

クライアント登録とユーザー無効化の画面です。本番の管理コンソールではありません。

ブラウザは admin トークンを持ちません。Next.js が `IDENTITY_ADMIN_TOKEN` で IdP の `/admin/api/*` を呼びます。先に IdP へ同じトークンを渡してください。

```powershell
copy .env.example .env.local
npm install
npm run dev
```

http://localhost:3002
