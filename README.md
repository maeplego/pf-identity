# pf-identity

P01 の Identity Provider 製品リポジトリです。**本番認証基盤の置き換えではありません。**

デプロイ単位（server / admin / sample-rp）はディレクトリで分けますが、Git は 1 本です。設計文書はワークスペース側の `project/portfolio-plan/identity-platform/DESIGN.md` にあります。

```
apps/server/    Authorization Server (Go)
apps/admin/     クライアント CRUD・ユーザー無効化・監査ログ
apps/sample-rp  接続確認用 RP（Next.js）
apps/e2e/       Playwright（ログイン、redirect 拒否、管理画面）
deploy/         ローカル Compose
```

メール検証は MVP では省略しています。そのためパスワード再設定などのアカウント復旧は弱いです。

## 開発（ホストで個別起動）

```powershell
cd apps/server
$env:IDENTITY_DEV_GENERATE_KEYS = "true"
$env:IDENTITY_STORE = "memory"
$env:IDENTITY_ADMIN_TOKEN = "change-me-locally"
$env:IDENTITY_SEED_PUBLIC_CLIENT_ID = "sample-rp"
$env:IDENTITY_SEED_PUBLIC_REDIRECT_URI = "http://localhost:3001/callback"
go test ./...
go run ./cmd/server
```

- sample-rp: `apps/sample-rp` で `npm install` のあと `npm run dev`（http://localhost:3001）
- admin: `apps/admin` で `.env.example` を `.env.local` にコピーし `npm run dev`（http://localhost:3002）

## Compose

```powershell
copy deploy/.env.example deploy/.env
docker compose -f deploy/compose.yaml --env-file deploy/.env up --build
```

- IdP: http://localhost:8080
- sample-rp: http://localhost:3001
- admin: http://localhost:3002

Postgres ストアのテストは `IDENTITY_TEST_DATABASE_URL` か Docker が必要です。どちらも無いときはそのパッケージだけ Skip します。

ブラウザ e2e は `apps/e2e`（手順はそちらの README）。同じ code の二回交換は `go test` 側が正本です。

## 他プロジェクトから接続する

各 RP はパスワードを持たず、この IdP の認可コード + PKCE S256 だけを使います。ユーザーの主キーは email ではなく ID Token の `sub` です。

1. 管理 UI（http://localhost:3002）でクライアントを作る。`redirect_uri` と `post_logout_redirect_uri` はクエリまで含めて **完全一致** で登録する（localhost のポート違いも別エントリ）。
2. confidential なら `client_secret` は作成時（または rotate）に一度だけ表示される。リポジトリに書かない。
3. public（PWA / モバイル）は secret を持たない。PKCE は必須。
4. RP は Discovery `http://localhost:8080/.well-known/openid-configuration` を読んでよい。
5. `/token` はサーバー側で交換する。ブラウザからの CORS は付けていない。
6. 発行クレーム: `iss`, `sub`, `aud`, `exp`, `iat`, `nonce`, `sid`、および scope に応じた `email` / `email_verified` / `name`。未検証メールは `email_verified=false`。
7. `state` と `nonce` は RP が発行し、コールバックと ID Token で照合する。実装の見本は `apps/sample-rp`。

スコープ初期セットは `openid`, `profile`, `email`, `offline_access`。アプリ固有 scope は Consent が崩れるので、利用側が必要になったら IdP に足す。
