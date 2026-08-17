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
        <button type="submit">作成</button>
      </form>
    </main>
  );
}
