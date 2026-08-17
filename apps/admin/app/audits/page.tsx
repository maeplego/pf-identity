import { listAudits, type Audit } from "../../lib/api";

export default async function AuditsPage({
  searchParams,
}: {
  searchParams: Promise<{ after?: string }>;
}) {
  const q = await searchParams;
  let audits: Audit[] = [];
  let next = "";
  let error = "";
  try {
    const page = await listAudits(q.after ?? "", 20);
    audits = page.items ?? [];
    next = page.next ?? "";
  } catch (e) {
    error = e instanceof Error ? e.message : "failed";
  }
  return (
    <main>
      <h1>監査ログ</h1>
      <p>パスワードやトークンは記録しません。新しい順です。</p>
      {error ? <p role="alert">{error}</p> : null}
      <table>
        <thead>
          <tr>
            <th>at</th>
            <th>type</th>
            <th>subject</th>
            <th>client</th>
            <th>ip</th>
            <th>note</th>
          </tr>
        </thead>
        <tbody>
          {audits.map((a) => (
            <tr key={a.id}>
              <td>{a.at}</td>
              <td>{a.type}</td>
              <td>{a.subject}</td>
              <td>{a.client_id}</td>
              <td>{a.ip}</td>
              <td>{a.note}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p>
        {q.after ? <a href="/audits">最新へ</a> : null}
        {next ? (
          <>
            {q.after ? " · " : null}
            <a href={`/audits?after=${encodeURIComponent(next)}`}>次のページ</a>
          </>
        ) : null}
      </p>
    </main>
  );
}
