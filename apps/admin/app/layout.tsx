import "./globals.css";

export const dynamic = "force-dynamic";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja">
      <body>
        <div className="site-shell">
          <header className="site-header">
            <div className="site-brand">
              <strong>pf-identity admin</strong>
              <span className="muted">学習用 IdP オペレーター</span>
            </div>
            <nav className="site-nav">
              <a href="/">ホーム</a>
              <a href="/clients">クライアント</a>
              <a href="/users">ユーザー</a>
              <a href="/audits">監査</a>
            </nav>
          </header>
          <main className="site-main">{children}</main>
        </div>
      </body>
    </html>
  );
}
