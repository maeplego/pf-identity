import { readCookie } from "../lib/cookies";
import { internalBase, rpLabel } from "../lib/env";
import { isRevoked } from "../lib/sids";

export default async function Home({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const q = await searchParams;
  const sid = await readCookie("rp_sid");
  const access = isRevoked(sid) ? undefined : await readCookie("rp_access");
  let userinfo: Record<string, unknown> | null = null;
  let userinfoError = "";
  if (access) {
    const res = await fetch(`${internalBase()}/userinfo`, {
      headers: { Authorization: `Bearer ${access}` },
      cache: "no-store",
    });
    if (res.ok) {
      userinfo = (await res.json()) as Record<string, unknown>;
    } else {
      userinfoError = `UserInfo ${res.status}`;
    }
  }

  return (
    <>
      <section className="hero">
        <h1 className="page-title" data-testid="rp-title">
          {rpLabel()}
        </h1>
        <p className="page-lead">学習用 RP です。authorization code + PKCE をサーバー側で交換します。</p>
      </section>

      {q.error ? (
        <p className="error" role="alert" data-testid="auth-error">
          エラー: {q.error}
        </p>
      ) : null}

      {userinfo ? (
        <div className="card stack">
          <h2 style={{ margin: 0 }}>UserInfo</h2>
          <pre data-testid="userinfo">{JSON.stringify(userinfo, null, 2)}</pre>
          <form action="/logout" method="post">
            <button type="submit" className="btn btn-secondary" data-testid="logout">
              ログアウト
            </button>
          </form>
        </div>
      ) : (
        <div className="card stack">
          {userinfoError ? <p className="error">{userinfoError}</p> : null}
          <p className="muted">IdP でログインして UserInfo を取得します。</p>
          <a href="/login" className="btn" data-testid="login-link">
            IdP でログイン
          </a>
        </div>
      )}
    </>
  );
}
