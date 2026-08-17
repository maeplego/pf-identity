import { NextRequest, NextResponse } from "next/server";

import { issuer } from "../../lib/env";
import { revokeSid } from "../../lib/sids";

export async function GET(req: NextRequest) {
  const iss = req.nextUrl.searchParams.get("iss") ?? "";
  const sid = req.nextUrl.searchParams.get("sid") ?? "";
  if (iss !== issuer() || !sid) {
    return new NextResponse("invalid front-channel logout", { status: 400 });
  }
  revokeSid(sid);
  return new NextResponse(null, { status: 204 });
}
