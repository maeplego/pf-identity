# sample-rp

P01 の接続確認用 Relying Party です。**本番アプリではありません。**

- `/login` で `state` / `nonce` / PKCE S256 を付けて IdP の `/authorize` へ送る
- `/callback` で認可コードを **このサーバーから** `/token` に渡す（ブラウザから token を叩かない）
- ID Token は JWKS で検証し、保存していた `nonce` と照合する
- ログアウトは IdP の `/end-session` に `id_token_hint` を付け、登録済み `post_logout_redirect_uri` へ戻る
- `/frontchannel-logout` は他 RP 経由の隠し iframe（`iss` と `sid`）を受け、セッションを無効化する
- `/backchannel-logout` は IdP からの `logout_token` POST を JWKS で検証し、`jti` の再利用を拒否してから同じ `sid` を無効化する

## 起動

IdP 側（`apps/server`）:

```powershell
$env:IDENTITY_DEV_GENERATE_KEYS = "true"
$env:IDENTITY_STORE = "memory"
$env:IDENTITY_SEED_PUBLIC_CLIENT_ID = "sample-rp"
$env:IDENTITY_SEED_PUBLIC_REDIRECT_URI = "http://localhost:3001/callback"
go run ./cmd/server
```

このアプリ:

```powershell
copy .env.example .env.local
npm install
npm run dev
```

ブラウザで http://localhost:3001 を開き、ログインする。
