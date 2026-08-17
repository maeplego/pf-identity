import { cookies } from "next/headers";

import { rotateSecret, updateClient } from "../../../lib/actions";
import { getClient } from "../../../lib/api";

export default async function ClientDetailPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { id } = await params;
  const q = await searchParams;
  const client = await getClient(id);
  const jar = await cookies();
  const secret = jar.get("flash_client_secret")?.value;
  return (
    <main>
      <h1>{client.id}</h1>
      {q.error ? <p role="alert">{q.error}</p> : null}
      {secret ? (
        <p role="status">
          client_secret（この画面だけ）: <code>{secret}</code>
        </p>
      ) : null}
      <form action={updateClient}>
        <input type="hidden" name="id" value={client.id} />
        <p>
          <label>
            name
            <br />
            <input name="name" defaultValue={client.name} required />
          </label>
        </p>
        <p>
          type: {client.type} / token_endpoint_auth: {client.token_endpoint_auth}
        </p>
        <p>
          <label>
            redirect_uris
            <br />
            <textarea name="redirect_uris" rows={4} defaultValue={client.redirect_uris.join("\n")} required />
          </label>
        </p>
        <button type="submit">更新</button>
      </form>
      {client.type === "confidential" ? (
        <form action={rotateSecret} style={{ marginTop: "1rem" }}>
          <input type="hidden" name="id" value={client.id} />
          <button type="submit">secret を再発行</button>
        </form>
      ) : null}
    </main>
  );
}
