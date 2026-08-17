import { cookies } from "next/headers";
import type { NextResponse } from "next/server";

const week = 60 * 60 * 24 * 7;

const base = {
  httpOnly: true,
  sameSite: "lax" as const,
  path: "/",
  secure: false,
};

// Route handlers must set cookies on the Response. cookies() from next/headers
// is dropped on NextResponse.redirect in the App Router.
export function setOn(res: NextResponse, name: string, value: string, maxAge = week) {
  res.cookies.set(name, value, { ...base, maxAge });
}

export function clearOn(res: NextResponse, name: string) {
  res.cookies.set(name, "", { ...base, maxAge: 0 });
}

export async function readCookie(name: string): Promise<string | undefined> {
  const jar = await cookies();
  return jar.get(name)?.value;
}
