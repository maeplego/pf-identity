# 管理 UI

クライアント登録、組織メンバー管理、ユーザー無効化の画面です。本番の管理コンソールではありません。

ブラウザは admin トークンを持ちません。Next.js が `IDENTITY_ADMIN_TOKEN` で IdP の `/admin/api/*` を呼びます。先に IdP へ同じトークンを渡してください。

| 画面 | 内容 |
| --- | --- |
| `/clients` | OAuth クライアント CRUD |
| `/orgs` | 組織一覧・作成 |
| `/orgs/[id]` | メンバー追加・role 変更・除名 |
| `/users` | ユーザー無効化 |
| `/audits` | 監査ログ |

```powershell
copy .env.example .env.local
npm install
npm run dev
```

http://localhost:3002
