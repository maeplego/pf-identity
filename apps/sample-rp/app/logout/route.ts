import { NextRequest, NextResponse } from "next/server";

import { clearCookie } from "../../lib/cookies";

export async function POST(req: NextRequest) {
  await clearCookie("rp_access");
  await clearCookie("rp_id");
  await clearCookie("rp_refresh");
  return NextResponse.redirect(new URL("/", req.url), { status: 303 });
}
