import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { adminFetch } from "./api";

export async function createClient(formData: FormData) {
  "use server";
  const redirectURIs = String(formData.get("redirect_uris") ?? "")
    .split("\n")
    .map((s) => s.trim())
    .filter(Boolean);
  const postLogoutURIs = String(formData.get("post_logout_redirect_uris") ?? "")
    .split("\n")
    .map((s) => s.trim())
    .filter(Boolean);
  const res = await adminFetch("/admin/api/clients", {
    method: "POST",
    body: JSON.stringify({
      id: String(formData.get("id") ?? "").trim(),
      name: String(formData.get("name") ?? "").trim(),
      type: String(formData.get("type") ?? "public"),
      redirect_uris: redirectURIs,
      post_logout_redirect_uris: postLogoutURIs,
      frontchannel_logout_uri: String(formData.get("frontchannel_logout_uri") ?? "").trim(),
      backchannel_logout_uri: String(formData.get("backchannel_logout_uri") ?? "").trim(),
    }),
  });
  if (!res.ok) {
    redirect(`/clients/new?error=${encodeURIComponent(await res.text())}`);
  }
  const body = (await res.json()) as { client: { id: string }; client_secret?: string };
  if (body.client_secret) {
    const jar = await cookies();
    jar.set("flash_client_secret", body.client_secret, {
      httpOnly: true,
      sameSite: "lax",
      path: "/",
      maxAge: 120,
    });
  }
  redirect(`/clients/${encodeURIComponent(body.client.id)}`);
}

export async function updateClient(formData: FormData) {
  "use server";
  const id = String(formData.get("id") ?? "");
  const redirectURIs = String(formData.get("redirect_uris") ?? "")
    .split("\n")
    .map((s) => s.trim())
    .filter(Boolean);
  const postLogoutURIs = String(formData.get("post_logout_redirect_uris") ?? "")
    .split("\n")
    .map((s) => s.trim())
    .filter(Boolean);
  const res = await adminFetch(`/admin/api/clients/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify({
      name: String(formData.get("name") ?? "").trim(),
      redirect_uris: redirectURIs,
      post_logout_redirect_uris: postLogoutURIs,
      frontchannel_logout_uri: String(formData.get("frontchannel_logout_uri") ?? "").trim(),
      backchannel_logout_uri: String(formData.get("backchannel_logout_uri") ?? "").trim(),
    }),
  });
  if (!res.ok) {
    redirect(`/clients/${encodeURIComponent(id)}?error=${encodeURIComponent(await res.text())}`);
  }
  redirect(`/clients/${encodeURIComponent(id)}`);
}

export async function rotateSecret(formData: FormData) {
  "use server";
  const id = String(formData.get("id") ?? "");
  const res = await adminFetch(`/admin/api/clients/${encodeURIComponent(id)}/rotate-secret`, {
    method: "POST",
  });
  if (!res.ok) {
    redirect(`/clients/${encodeURIComponent(id)}?error=${encodeURIComponent(await res.text())}`);
  }
  const body = (await res.json()) as { client_secret: string };
  const jar = await cookies();
  jar.set("flash_client_secret", body.client_secret, {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: 120,
  });
  redirect(`/clients/${encodeURIComponent(id)}`);
}

export async function setUserDisabled(formData: FormData) {
  "use server";
  const id = String(formData.get("id") ?? "");
  const disabled = String(formData.get("disabled") ?? "") === "true";
  const res = await adminFetch(`/admin/api/users/${encodeURIComponent(id)}/disabled`, {
    method: "POST",
    body: JSON.stringify({ disabled }),
  });
  if (!res.ok) {
    redirect(`/?error=${encodeURIComponent(await res.text())}`);
  }
  redirect("/users");
}
