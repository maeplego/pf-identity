# pf-identity

P01 の Identity Provider 製品リポジトリです。**本番認証基盤の置き換えではありません。**

デプロイ単位（server / admin / sample-rp）はディレクトリで分けますが、Git は 1 本です。設計文書はワークスペース側の `project/portfolio-plan/identity-platform/DESIGN.md` にあります。

```
apps/server/   Authorization Server (Go)
apps/admin/    未作成
apps/sample-rp 未作成
deploy/        ローカル Compose
```

## 開発

```powershell
cd apps/server
$env:IDENTITY_DEV_GENERATE_KEYS = "true"
$env:IDENTITY_STORE = "memory"
go test ./...
go run ./cmd/server
```

Postgres が必要になったら `deploy/.env.example` をコピーして `docker compose -f deploy/compose.yaml --env-file deploy/.env up -d db` です。
