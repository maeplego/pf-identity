import { cookies } from "next/headers";

const week = 60 * 60 * 24 * 7;

export async function setHttpOnly(name: string, value: string, maxAge = week) {
  const jar = await cookies();
  jar.set(name, value, {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge,
    secure: false,
  });
}

export async function clearCookie(name: string) {
  const jar = await cookies();
  jar.delete(name);
}

export async function readCookie(name: string): Promise<string | undefined> {
  const jar = await cookies();
  return jar.get(name)?.value;
}
