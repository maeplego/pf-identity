import { NextRequest, NextResponse } from "next/server";

import { clearOn } from "../../lib/cookies";

export async function GET(req: NextRequest) {
  const state = req.nextUrl.searchParams.get("state") ?? "";
  const expected = req.cookies.get("rp_logout_state")?.value ?? "";
  if (!expected || state !== expected) {
    const res = NextResponse.redirect(new URL("/?error=logout_state", req.url), { status: 303 });
    clearOn(res, "rp_logout_state");
    return res;
  }
  const res = NextResponse.redirect(new URL("/", req.url), { status: 303 });
  clearOn(res, "rp_access");
  clearOn(res, "rp_id");
  clearOn(res, "rp_refresh");
  clearOn(res, "rp_sid");
  clearOn(res, "rp_logout_state");
  return res;
}
