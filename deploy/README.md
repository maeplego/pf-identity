# Local compose for P01. Not a production deployment.

Postgres is for the durable store. Set `IDENTITY_STORE=postgres` and `IDENTITY_DATABASE_URL` on `apps/server` after this database is up.

Copy `.env.example` to `.env` (gitignored) and start from repo root:

```powershell
docker compose -f deploy/compose.yaml --env-file deploy/.env up -d db
```

Do not reuse the example password outside this machine.
