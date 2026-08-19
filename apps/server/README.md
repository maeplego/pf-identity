# Identity Provider（server）

学習用の OpenID Connect 認可サーバーです。認可コード + PKCE、サーバーサイドセッション、RS256 の ID Token を出します。パスワード・認可コード・リフレッシュトークンはハッシュして保存します。リポジトリルートの README も見てください。

implicit、リソースオーナーパスワード、SAML、ソーシャルログインはありません。

```powershell
copy .env.example .env
go test ./...
go run ./cmd/server
```

既定の issuer は `http://localhost:8080` です。Discovery は `/.well-known/openid-configuration` です。秘密は環境変数だけにしてください。

`redirect_uri` は登録値と完全一致です。公開クライアントは PKCE S256 必須です。Cookie は HttpOnly です。
