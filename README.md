# pf-identity

P01 の Identity Provider 製品リポジトリです。**本番認証基盤の置き換えではありません。**

デプロイ単位（server / admin / sample-rp）はディレクトリで分けますが、Git は 1 本です。設計文書はワークスペース側の `project/portfolio-plan/identity-platform/DESIGN.md` にあります。

```
apps/server/   Authorization Server (Go)
apps/admin/    未作成
apps/sample-rp 接続確認用 RP（Next.js）
deploy/        ローカル Compose
```

## 開発

```powershell
cd apps/server
$env:IDENTITY_DEV_GENERATE_KEYS = "true"
$env:IDENTITY_STORE = "memory"
$env:IDENTITY_SEED_PUBLIC_CLIENT_ID = "sample-rp"
$env:IDENTITY_SEED_PUBLIC_REDIRECT_URI = "http://localhost:3001/callback"
go test ./...
go run ./cmd/server
```

sample-rp は `apps/sample-rp` で `npm install` のあと `npm run dev`（ポート 3001）。手順はそちらの README。

Postgres が必要になったら `deploy/.env.example` をコピーして `docker compose -f deploy/compose.yaml --env-file deploy/.env up -d db` です。サーバーは `IDENTITY_STORE=postgres` と `IDENTITY_DATABASE_URL` を設定する。
Postgres ストアのテストは `IDENTITY_TEST_DATABASE_URL` か Docker が必要です。どちらも無いときはそのパッケージだけ Skip します。
