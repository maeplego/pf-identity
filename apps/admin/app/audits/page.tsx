import { listAudits, type Audit } from "../../lib/api";

export default async function AuditsPage() {
  let audits: Audit[] = [];
  let error = "";
  try {
    audits = await listAudits();
  } catch (e) {
    error = e instanceof Error ? e.message : "failed";
  }
  return (
    <main>
      <h1>監査ログ</h1>
      <p>パスワードやトークンは記録しません。</p>
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
    </main>
  );
}
