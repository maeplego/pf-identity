import { NextRequest, NextResponse } from "next/server";

import { clearOn } from "../../lib/cookies";

export async function POST(req: NextRequest) {
  const res = NextResponse.redirect(new URL("/", req.url), { status: 303 });
  clearOn(res, "rp_access");
  clearOn(res, "rp_id");
  clearOn(res, "rp_refresh");
  return res;
}
