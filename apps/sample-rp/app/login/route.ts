import { NextResponse } from "next/server";

import { setHttpOnly } from "../../lib/cookies";
import { clientId, issuer, redirectUri } from "../../lib/env";
import { randomString, s256 } from "../../lib/pkce";

export async function GET() {
  const state = randomString(16);
  const nonce = randomString(16);
  const verifier = randomString(32);
  await setHttpOnly("rp_state", state, 600);
  await setHttpOnly("rp_nonce", nonce, 600);
  await setHttpOnly("rp_verifier", verifier, 600);

  const q = new URLSearchParams({
    response_type: "code",
    client_id: clientId(),
    redirect_uri: redirectUri(),
    scope: "openid profile email offline_access",
    state,
    nonce,
    code_challenge: s256(verifier),
    code_challenge_method: "S256",
  });
  return NextResponse.redirect(`${issuer()}/authorize?${q.toString()}`);
}
