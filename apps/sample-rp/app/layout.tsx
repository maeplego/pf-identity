import "./globals.css";

export const dynamic = "force-dynamic";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja">
      <body>
        <div className="site-shell">
          <header className="site-header">
            <div className="site-brand">
              <strong>pf-identity sample RP</strong>
              <span className="muted">authorization code + PKCE</span>
            </div>
            <nav className="site-nav">
              <a href="/">ホーム</a>
            </nav>
          </header>
          <main className="site-main">{children}</main>
        </div>
      </body>
    </html>
  );
}
