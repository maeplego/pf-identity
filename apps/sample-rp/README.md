# sample-rp

Identity Provider への接続確認用 Relying Party です。本番アプリではありません。

- `/login` で `state` / `nonce` / PKCE を付けて認可へ送る
- `/callback` で認可コードを **このサーバーから** `/token` に渡す（ブラウザから token は叩かない）
- ID Token を JWKS で検証し、保存していた `nonce` と照合する
- ログアウトは IdP の `/end-session` へ。他 RP とは Front-Channel / Back-Channel でセッションを揃える

## 起動

先に `apps/server` を開発キーと sample-rp 向けシードで起動し、こちらで:

```powershell
copy .env.example .env.local
npm install
npm run dev
```

http://localhost:3001 でログインします。Compose では 2 つ目の RP が http://localhost:3003 です。
