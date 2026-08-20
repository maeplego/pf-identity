export default function Home() {
  return (
    <>
      <section className="hero">
        <h1 className="page-title">管理 UI</h1>
        <p className="page-lead">
          学習用 IdP のオペレーター画面です。トークンはサーバー側だけに置き、ブラウザへは出しません。
        </p>
      </section>
      <div className="card-grid">
        <a className="card" href="/clients">
          <strong>クライアント CRUD</strong>
          <p className="muted">OAuth クライアントの登録・更新・削除</p>
        </a>
        <a className="card" href="/users">
          <strong>ユーザーの無効化</strong>
          <p className="muted">アカウント状態の管理</p>
        </a>
        <a className="card" href="/audits">
          <strong>監査ログ</strong>
          <p className="muted">オペレーション履歴の参照</p>
        </a>
      </div>
    </>
  );
}
