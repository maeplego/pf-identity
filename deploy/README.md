# ローカル Compose（P01）

Postgres、IdP、管理 UI、sample-rp、sample-rp-b（SSO とログアウト連動）を起動します。本番デプロイではありません。

リポジトリルートから:

```powershell
copy deploy/.env.example deploy/.env
docker compose -f deploy/compose.yaml --env-file deploy/.env up --build
```

`.env.example` のパスワードや admin トークンを、このマシン以外で使い回さないでください。ブラウザは `localhost`、コンテナ同士は `http://idp:8080` です。
