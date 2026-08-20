import Link from "next/link";

import { addOrganizationMember, removeOrganizationMember, updateOrganizationMemberRole } from "../../../lib/actions";
import { getOrganization, listOrganizationMembers, type OrgMember, type Organization } from "../../../lib/api";

export default async function OrgDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { id } = await params;
  const sp = await searchParams;
  let org: Organization | null = null;
  let members: OrgMember[] = [];
  let error = sp.error || "";
  try {
    org = await getOrganization(id);
    members = await listOrganizationMembers(id);
  } catch (e) {
    error = e instanceof Error ? e.message : "failed";
  }
  const ownerCount = members.filter((m) => m.role === "owner").length;

  return (
    <main>
      <p className="muted">
        <Link href="/orgs">← 組織一覧</Link>
      </p>
      <h1>{org?.name || "組織"}</h1>
      {org ? <p className="muted">id: {org.id}</p> : null}
      {error ? <p role="alert">{error}</p> : null}

      <section style={{ marginBottom: "1.5rem" }}>
        <h2 style={{ fontSize: "1.1rem" }}>メンバー追加</h2>
        <form action={addOrganizationMember} style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem" }}>
          <input type="hidden" name="org_id" value={id} />
          <input name="email" type="email" placeholder="email" required style={{ minWidth: 220 }} />
          <select name="role" defaultValue="member">
            <option value="member">member</option>
            <option value="owner">owner</option>
          </select>
          <button type="submit">追加</button>
        </form>
      </section>

      <table>
        <thead>
          <tr>
            <th>email</th>
            <th>name</th>
            <th>role</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {members.map((m) => (
            <tr key={m.user_id}>
              <td>{m.email || m.user_id}</td>
              <td>{m.name}</td>
              <td>
                <form action={updateOrganizationMemberRole} style={{ display: "inline-flex", gap: "0.35rem" }}>
                  <input type="hidden" name="org_id" value={id} />
                  <input type="hidden" name="user_id" value={m.user_id} />
                  <select name="role" defaultValue={m.role}>
                    <option value="member">member</option>
                    <option value="owner">owner</option>
                  </select>
                  <button type="submit">変更</button>
                </form>
              </td>
              <td>
                {m.role === "owner" && ownerCount <= 1 ? (
                  <span className="muted">唯一の owner</span>
                ) : (
                  <form action={removeOrganizationMember}>
                    <input type="hidden" name="org_id" value={id} />
                    <input type="hidden" name="user_id" value={m.user_id} />
                    <button type="submit">除名</button>
                  </form>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </main>
  );
}
