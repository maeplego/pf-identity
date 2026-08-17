import { createClient } from "../../../lib/actions";

export default async function NewClientPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const q = await searchParams;
  return (
    <main>
      <h1>クライアント登録</h1>
      {q.error ? <p role="alert">{q.error}</p> : null}
      <form action={createClient}>
        <p>
          <label>
            id（空なら自動）
            <br />
            <input name="id" placeholder="blog-cms" />
          </label>
        </p>
        <p>
          <label>
            name
            <br />
            <input name="name" required />
          </label>
        </p>
        <p>
          <label>
            type
            <br />
            <select name="type" defaultValue="public">
              <option value="public">public（PKCE、secret なし）</option>
              <option value="confidential">confidential（secret は一度だけ表示）</option>
            </select>
          </label>
        </p>
        <p>
          <label>
            redirect_uris（1 行に 1 つ。完全一致）
            <br />
            <textarea name="redirect_uris" rows={4} required placeholder="http://localhost:3000/callback" />
          </label>
        </p>
        <p>
          <label>
            post_logout_redirect_uris（任意。RP-Initiated Logout の戻り先）
            <br />
            <textarea name="post_logout_redirect_uris" rows={3} placeholder="http://localhost:3000/logged-out" />
          </label>
        </p>
        <p>
          <label>
            frontchannel_logout_uri（任意。他 RP ログアウト時に iframe で叩かれる）
            <br />
            <input name="frontchannel_logout_uri" placeholder="http://localhost:3000/frontchannel-logout" />
          </label>
        </p>
        <button type="submit">作成</button>
      </form>
    </main>
  );
}
