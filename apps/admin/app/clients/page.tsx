import { listClients, type Client } from "../../lib/api";

export default async function ClientsPage() {
  let clients: Client[] = [];
  let error = "";
  try {
    clients = await listClients();
  } catch (e) {
    error = e instanceof Error ? e.message : "failed";
  }
  return (
    <main>
      <h1>クライアント</h1>
      <p>
        <a href="/clients/new">新規登録</a>
      </p>
      {error ? <p role="alert">{error}</p> : null}
      <table>
        <thead>
          <tr>
            <th>id</th>
            <th>name</th>
            <th>type</th>
          </tr>
        </thead>
        <tbody>
          {clients.map((c) => (
            <tr key={c.id}>
              <td>
                <a href={`/clients/${c.id}`}>{c.id}</a>
              </td>
              <td>{c.name}</td>
              <td>{c.type}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </main>
  );
}
