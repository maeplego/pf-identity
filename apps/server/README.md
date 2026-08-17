# pf-identity server

学習用の OpenID Connect Identity Provider です。**本番の認証基盤の置き換えではありません。**

このモジュールは製品リポジトリ `pf-identity` の `apps/server` です。リポジトリルートの README も参照してください。

Authorization Server として、認可コード + PKCE、サーバーサイドセッション、RS256 の ID Token を提供します。パスワード・認可コード・リフレッシュトークンはハッシュして保存します。

## 範囲外（意図的）

- implicit / resource owner password
- SAML、CIBA、PAR
- ソーシャルログイン

## 開発

秘密は環境変数で渡します。リポジトリに PEM や `.env` を置かないでください。

```bash
copy .env.example .env   # 値はローカルだけ
go test ./...
go run ./cmd/server
```

既定の issuer は `http://localhost:8080` です。Discovery は `/.well-known/openid-configuration`。

永続化は `IDENTITY_STORE=postgres` と `IDENTITY_DATABASE_URL`。開発用 public クライアントは `IDENTITY_SEED_PUBLIC_CLIENT_ID` と `IDENTITY_SEED_PUBLIC_REDIRECT_URI`（`post_logout_redirect_uri` は未指定なら `/callback` を `/logged-out` に置き換える）。管理 API は `IDENTITY_ADMIN_TOKEN`（未設定なら 401）。

RP-Initiated Logout は Discovery の `end_session_endpoint`（`/end-session`）。`post_logout_redirect_uri` は登録値と完全一致。`id_token_hint` が現在のセッションと一致すれば確認画面を出さない。同じ OP セッションでログインした RP には Front-Channel Logout（隠し iframe、`iss` と `sid`）と Back-Channel Logout（`logout_token` の POST）を送る。Discovery は `frontchannel_logout_supported` と `backchannel_logout_supported` を返す。

## セキュリティ上の前提

- `redirect_uri` は登録値と完全一致（クエリの追加も拒否）
- public クライアントは PKCE S256 必須
- Cookie は HttpOnly。本番では `IDENTITY_COOKIE_SECURE=true`
