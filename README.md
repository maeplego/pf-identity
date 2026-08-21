# pf-identity

| まず | リンク |
| --- | --- |
| 採用の位置づけ | [HIRING.md](https://github.com/maeplego/portfolio-plan/blob/master/portfolio-plan/HIRING.md) |
| 確認手順 | [REVIEW.md](https://github.com/maeplego/portfolio-plan/blob/master/portfolio-plan/REVIEW.md) |

学習用の OpenID Connect Identity Provider です。認可コード + PKCE、同意画面、トークン発行、JWKS、ログアウト連動までを実装しています。**本番の認証基盤の置き換えではありません。**

| ディレクトリ | 役割 |
| --- | --- |
| `apps/server` | 認可サーバー（Go） |
| `apps/admin` | クライアント登録とユーザー無効化 |
| `apps/sample-rp` | 接続確認用の Relying Party（Next.js） |
| `apps/e2e` | ブラウザテスト（Playwright） |
| `deploy/` | ローカル Docker Compose |

メール検証は入れていません。パスワード再設定などのアカウント復旧は弱いです。

## 起動（Compose）

```powershell
copy deploy/.env.example deploy/.env
docker compose -f deploy/compose.yaml --env-file deploy/.env up --build
```

| URL | 用途 |
| --- | --- |
| http://localhost:8080 | Identity Provider |
| http://localhost:3001 | sample-rp（ログイン確認） |
| http://localhost:3003 | 2 つ目の RP（片方でログアウトすると、もう片方も落ちる） |
| http://localhost:3002 | 管理画面 |

デモユーザーは Compose が用意します。本番アカウントではありません。Discovery は `http://localhost:8080/.well-known/openid-configuration` です。

## テスト

```powershell
cd apps/server
go test ./...
```

Postgres 向けのテストは、接続先が無いときは skip します。ブラウザ e2e は `apps/e2e` です（`npx playwright test`。GitHub では `workflow_dispatch`）。同じ認可コードを二回使う拒否は `go test` が正です。

Compose 起動後のヘルス:

```powershell
node scripts/compose-smoke.mjs http://localhost:8080/health
```

## 他サービスから接続する

アプリ側はパスワードを持ちません。認可コード + PKCE（S256）だけを使います。ユーザーの主キーはメールではなく ID Token の `sub` です。

1. 管理画面でクライアントを作る。`redirect_uri` と `post_logout_redirect_uri` は、クエリまで含めて登録値と完全一致させます。
2. 秘密のあるクライアントは、作成時（または rotate 時）に一度だけ `client_secret` を表示します。Git に書かないでください。
3. 公開クライアント（ブラウザ / モバイル）は secret を持ちません。PKCE は必須です。
4. `/token` はサーバー側で交換します。ブラウザからの CORS は付けていません。
5. `state` と `nonce` は RP が発行し、コールバックと ID Token で照合します。見本は `apps/sample-rp` です。

初期スコープは `openid` / `profile` / `email` / `offline_access` です。

設計の詳細は [portfolio-plan](https://github.com/maeplego/portfolio-plan) の `portfolio-plan/identity-platform/docs/` です。

## ライセンスと利用条件

本リポジトリは **デモ・学習・社内評価用** です。現状品質に **保証はありません**。

- 許可: クローン、ローカル実行、学習、非本番の評価
- 別契約が必要: 本番運用、有償サービスへの組込み、再販・托管の提供

詳細は [LICENSE](./LICENSE) と [licensing.md](https://github.com/maeplego/portfolio-plan/blob/master/portfolio-plan/licensing.md) を参照してください。

