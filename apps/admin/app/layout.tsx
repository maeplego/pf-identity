export const dynamic = "force-dynamic";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja">
      <body style={{ fontFamily: "sans-serif", margin: 0 }}>
        <header style={{ padding: "1rem 2rem", borderBottom: "1px solid #ddd" }}>
          <strong>pf-identity admin</strong>
          <nav style={{ display: "inline", marginLeft: "1.5rem" }}>
            <a href="/clients" style={{ marginRight: "1rem" }}>
              クライアント
            </a>
            <a href="/users" style={{ marginRight: "1rem" }}>
              ユーザー
            </a>
            <a href="/audits">監査</a>
          </nav>
        </header>
        <div style={{ maxWidth: 880, margin: "2rem auto", padding: "0 1rem" }}>{children}</div>
      </body>
    </html>
  );
}
