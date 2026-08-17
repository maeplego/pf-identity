export default function Home() {
  return (
    <main>
      <h1>管理 UI</h1>
      <p>学習用 IdP のオペレーター画面です。トークンはサーバー側だけに置き、ブラウザへは出しません。</p>
      <ul>
        <li>
          <a href="/clients">クライアント CRUD</a>
        </li>
        <li>
          <a href="/users">ユーザーの無効化</a>
        </li>
        <li>
          <a href="/audits">監査ログ</a>
        </li>
      </ul>
    </main>
  );
}
