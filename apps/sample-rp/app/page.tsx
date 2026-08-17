import { readCookie } from "../lib/cookies";
import { issuer } from "../lib/env";

export default async function Home({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const q = await searchParams;
  const access = await readCookie("rp_access");
  let userinfo: Record<string, unknown> | null = null;
  let userinfoError = "";
  if (access) {
    const res = await fetch(`${issuer()}/userinfo`, {
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
    <main style={{ fontFamily: "sans-serif", maxWidth: 640, margin: "2rem auto" }}>
      <h1>sample-rp</h1>
      <p>学習用 RP です。authorization code + PKCE をサーバー側で交換します。</p>
      {q.error ? <p role="alert">エラー: {q.error}</p> : null}
      {userinfo ? (
        <>
          <h2>UserInfo</h2>
          <pre>{JSON.stringify(userinfo, null, 2)}</pre>
          <form action="/logout" method="post">
            <button type="submit">ログアウト</button>
          </form>
        </>
      ) : (
        <>
          {userinfoError ? <p>{userinfoError}</p> : null}
          <p>
            <a href="/login">IdP でログイン</a>
          </p>
        </>
      )}
    </main>
  );
}
