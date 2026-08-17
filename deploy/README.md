# Local compose for P01. Not a production deployment.

Starts Postgres, the IdP, admin UI, sample-rp, and sample-rp-b (2 つの RP で SSO / ログアウト連動デモ).

Copy `.env.example` to `.env` (gitignored) and from the product repo root:

```powershell
docker compose -f deploy/compose.yaml --env-file deploy/.env up --build
```

Do not reuse the example passwords or admin token outside this machine.

Browser URLs use `localhost` (issuer and redirects). Containers call the IdP as `http://idp:8080`.
