import Link from "next/link";

import { createOrganization } from "../../lib/actions";
import { listOrganizations, type Organization } from "../../lib/api";

export default async function OrgsPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const sp = await searchParams;
  let orgs: Organization[] = [];
  let error = sp.error || "";
  try {
    orgs = await listOrganizations();
  } catch (e) {
    error = e instanceof Error ? e.message : "failed";
  }
  return (
    <main>
      <h1>組織</h1>
      <p className="muted">テナント（organization）の一覧とメンバー管理です。</p>
      {error ? <p role="alert">{error}</p> : null}

      <section style={{ marginBottom: "2rem" }}>
        <h2 style={{ fontSize: "1.1rem" }}>新規作成</h2>
        <form action={createOrganization} className="row" style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem" }}>
          <input name="name" placeholder="組織名" required style={{ minWidth: 180 }} />
          <input name="owner_email" type="email" placeholder="owner の email" required style={{ minWidth: 220 }} />
          <button type="submit">作成</button>
        </form>
      </section>

      <table>
        <thead>
          <tr>
            <th>name</th>
            <th>id</th>
            <th>created</th>
          </tr>
        </thead>
        <tbody>
          {orgs.map((o) => (
            <tr key={o.id}>
              <td>
                <Link href={`/orgs/${encodeURIComponent(o.id)}`}>{o.name}</Link>
              </td>
              <td className="muted">{o.id}</td>
              <td>{o.created_at}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </main>
  );
}
