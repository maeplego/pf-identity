import { createRemoteJWKSet, jwtVerify } from "jose";

import { clientId, issuer } from "./env";

export async function verifyIdToken(idToken: string, nonce: string) {
  const iss = issuer();
  const JWKS = createRemoteJWKSet(new URL(`${iss}/jwks.json`));
  const { payload } = await jwtVerify(idToken, JWKS, {
    issuer: iss,
    audience: clientId(),
  });
  if (payload.nonce !== nonce) {
    throw new Error("nonce mismatch");
  }
  return payload;
}
