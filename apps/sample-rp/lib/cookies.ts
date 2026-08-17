import { cookies } from "next/headers";
import type { NextRequest, NextResponse } from "next/server";

import { clientId } from "./env";

const week = 60 * 60 * 24 * 7;

const base = {
  httpOnly: true,
  sameSite: "lax" as const,
  path: "/",
  secure: false,
};

// localhost の複数 RP は port が違っても Cookie が共有されるので、クライアントごとに名前を分ける。
export function cookieKey(name: string): string {
  return `${name}_${clientId().replace(/-/g, "_")}`;
}

// Route handlers must set cookies on the Response. cookies() from next/headers
// is dropped on NextResponse.redirect in the App Router.
export function setOn(res: NextResponse, name: string, value: string, maxAge = week) {
  res.cookies.set(cookieKey(name), value, { ...base, maxAge });
}

export function clearOn(res: NextResponse, name: string) {
  res.cookies.set(cookieKey(name), "", { ...base, maxAge: 0 });
}

export async function readCookie(name: string): Promise<string | undefined> {
  const jar = await cookies();
  return jar.get(cookieKey(name))?.value;
}

export function readRequestCookie(req: NextRequest, name: string): string | undefined {
  return req.cookies.get(cookieKey(name))?.value;
}
