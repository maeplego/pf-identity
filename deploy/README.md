# Local compose for P01. Not a production deployment.

Postgres is for the durable store. `apps/server` still defaults to `IDENTITY_STORE=memory`.

Copy `.env.example` to `.env` (gitignored) and start from repo root:

```powershell
docker compose -f deploy/compose.yaml --env-file deploy/.env up -d db
```

Do not reuse the example password outside this machine.
