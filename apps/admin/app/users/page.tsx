import { setUserDisabled } from "../../lib/actions";
import { listUsers, type User } from "../../lib/api";

export default async function UsersPage() {
  let users: User[] = [];
  let error = "";
  try {
    users = await listUsers();
  } catch (e) {
    error = e instanceof Error ? e.message : "failed";
  }
  return (
    <main>
      <h1>ユーザー</h1>
      {error ? <p role="alert">{error}</p> : null}
      <table>
        <thead>
          <tr>
            <th>email</th>
            <th>name</th>
            <th>disabled</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.id}>
              <td>{u.email}</td>
              <td>{u.name}</td>
              <td>{u.disabled ? "yes" : "no"}</td>
              <td>
                <form action={setUserDisabled}>
                  <input type="hidden" name="id" value={u.id} />
                  <input type="hidden" name="disabled" value={u.disabled ? "false" : "true"} />
                  <button type="submit">{u.disabled ? "有効化" : "無効化"}</button>
                </form>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </main>
  );
}
