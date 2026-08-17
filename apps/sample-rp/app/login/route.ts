import { NextResponse } from "next/server";

import { setOn } from "../../lib/cookies";
import { clientId, issuer, redirectUri } from "../../lib/env";
import { randomString, s256 } from "../../lib/pkce";

export async function GET() {
  const state = randomString(16);
  const nonce = randomString(16);
  const verifier = randomString(32);
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
  const res = NextResponse.redirect(`${issuer()}/authorize?${q.toString()}`);
  setOn(res, "rp_state", state, 600);
  setOn(res, "rp_nonce", nonce, 600);
  setOn(res, "rp_verifier", verifier, 600);
  return res;
}
