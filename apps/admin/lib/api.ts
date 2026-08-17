function apiBase(): string {
  const v = process.env.IDENTITY_API_BASE?.replace(/\/$/, "");
  if (!v) {
    throw new Error("IDENTITY_API_BASE is required");
  }
  return v;
}

function adminToken(): string {
  const v = process.env.IDENTITY_ADMIN_TOKEN;
  if (!v) {
    throw new Error("IDENTITY_ADMIN_TOKEN is required");
  }
  return v;
}

export type Client = {
  id: string;
  name: string;
  type: string;
  redirect_uris: string[];
  token_endpoint_auth: string;
  has_secret: boolean;
};

export type User = {
  id: string;
  email: string;
  name: string;
  disabled: boolean;
  created_at: string;
};

export type Audit = {
  id: string;
  type: string;
  at: string;
  subject: string;
  client_id: string;
  ip: string;
  note: string;
};

export async function adminFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("Authorization", `Bearer ${adminToken()}`);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(`${apiBase()}${path}`, { ...init, headers, cache: "no-store" });
}

export async function listClients(): Promise<Client[]> {
  const res = await adminFetch("/admin/api/clients");
  if (!res.ok) {
    throw new Error(`list clients ${res.status}`);
  }
  return (await res.json()) as Client[];
}

export async function getClient(id: string): Promise<Client> {
  const res = await adminFetch(`/admin/api/clients/${encodeURIComponent(id)}`);
  if (!res.ok) {
    throw new Error(`get client ${res.status}`);
  }
  return (await res.json()) as Client;
}

export async function listUsers(): Promise<User[]> {
  const res = await adminFetch("/admin/api/users");
  if (!res.ok) {
    throw new Error(`list users ${res.status}`);
  }
  return (await res.json()) as User[];
}

export async function listAudits(): Promise<Audit[]> {
  const res = await adminFetch("/admin/api/audits");
  if (!res.ok) {
    throw new Error(`list audits ${res.status}`);
  }
  return (await res.json()) as Audit[];
}
