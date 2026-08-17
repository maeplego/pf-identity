import { NextRequest, NextResponse } from "next/server";

import { clearCookie, readCookie, setHttpOnly } from "../../lib/cookies";
import { clientId, issuer, redirectUri } from "../../lib/env";
import { verifyIdToken } from "../../lib/idtoken";

export async function GET(req: NextRequest) {
  const url = req.nextUrl;
  const err = url.searchParams.get("error");
  if (err) {
    return NextResponse.redirect(new URL(`/?error=${encodeURIComponent(err)}`, url.origin));
  }
  const code = url.searchParams.get("code") ?? "";
  const state = url.searchParams.get("state") ?? "";
  const expected = await readCookie("rp_state");
  const nonce = await readCookie("rp_nonce");
  const verifier = await readCookie("rp_verifier");
  if (!code || !state || !expected || state !== expected || !nonce || !verifier) {
    return NextResponse.redirect(new URL("/?error=state", url.origin));
  }

  const body = new URLSearchParams({
    grant_type: "authorization_code",
    client_id: clientId(),
    code,
    redirect_uri: redirectUri(),
    code_verifier: verifier,
  });
  const tokenRes = await fetch(`${issuer()}/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body,
    cache: "no-store",
  });
  if (!tokenRes.ok) {
    return NextResponse.redirect(new URL("/?error=token", url.origin));
  }
  const tokens = (await tokenRes.json()) as {
    access_token?: string;
    id_token?: string;
    refresh_token?: string;
  };
  if (!tokens.access_token || !tokens.id_token) {
    return NextResponse.redirect(new URL("/?error=token", url.origin));
  }
  try {
    await verifyIdToken(tokens.id_token, nonce);
  } catch {
    return NextResponse.redirect(new URL("/?error=id_token", url.origin));
  }

  await setHttpOnly("rp_access", tokens.access_token);
  await setHttpOnly("rp_id", tokens.id_token);
  if (tokens.refresh_token) {
    await setHttpOnly("rp_refresh", tokens.refresh_token);
  }
  await clearCookie("rp_state");
  await clearCookie("rp_nonce");
  await clearCookie("rp_verifier");
  return NextResponse.redirect(new URL("/", url.origin));
}
