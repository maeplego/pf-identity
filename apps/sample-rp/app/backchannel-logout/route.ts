import { jwtVerify, createRemoteJWKSet } from "jose";
import { NextRequest, NextResponse } from "next/server";

import { clientId, internalBase, issuer } from "../../lib/env";
import { consumeJti, revokeSid } from "../../lib/sids";

const logoutEvent = "http://schemas.openid.net/event/backchannel-logout";

export async function POST(req: NextRequest) {
  const contentType = req.headers.get("content-type") ?? "";
  if (
    !contentType.includes("application/x-www-form-urlencoded") &&
    !contentType.includes("multipart/form-data")
  ) {
    return new NextResponse("missing logout_token", { status: 400 });
  }
  let logoutToken = "";
  try {
    const form = await req.formData();
    logoutToken = String(form.get("logout_token") ?? "");
  } catch {
    return new NextResponse("missing logout_token", { status: 400 });
  }
  if (!logoutToken) {
    return new NextResponse("missing logout_token", { status: 400 });
  }
  try {
    const JWKS = createRemoteJWKSet(new URL(`${internalBase()}/jwks.json`));
    const { payload } = await jwtVerify(logoutToken, JWKS, {
      issuer: issuer(),
      audience: clientId(),
    });
    if (payload.nonce !== undefined) {
      return new NextResponse("nonce is not allowed in logout_token", { status: 400 });
    }
    const events = payload.events;
    if (!events || typeof events !== "object" || !(logoutEvent in events)) {
      return new NextResponse("invalid events", { status: 400 });
    }
    const jti = typeof payload.jti === "string" ? payload.jti : "";
    if (!consumeJti(jti)) {
      return new NextResponse("replayed or missing jti", { status: 400 });
    }
    const sid = typeof payload.sid === "string" ? payload.sid : "";
    if (sid) {
      revokeSid(sid);
    }
    return new NextResponse(null, { status: 204 });
  } catch {
    return new NextResponse("invalid logout_token", { status: 400 });
  }
}
