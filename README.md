# pf-identity-server

学習用の OpenID Connect Identity Provider です。**本番の認証基盤の置き換えではありません。**

Authorization Server として、認可コード + PKCE、サーバーサイドセッション、RS256 の ID Token を提供します。パスワード・認可コード・リフレッシュトークンはハッシュして保存します。

## 範囲外（意図的）

- implicit / resource owner password
- SAML、CIBA、PAR
- 完全な RP-initiated logout
- ソーシャルログイン

## 開発

秘密は環境変数で渡します。リポジトリに PEM や `.env` を置かないでください。

```bash
copy .env.example .env   # 値はローカルだけ
go test ./...
go run ./cmd/server
```

既定の issuer は `http://localhost:8080` です。Discovery は `/.well-known/openid-configuration`。

## セキュリティ上の前提

- `redirect_uri` は登録値と完全一致（クエリの追加も拒否）
- public クライアントは PKCE S256 必須
- Cookie は HttpOnly。本番では `IDENTITY_COOKIE_SECURE=true`
