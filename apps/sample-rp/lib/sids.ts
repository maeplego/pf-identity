const revoked = new Set<string>();

export function rememberSid(sid: string) {
  if (sid) {
    revoked.delete(sid);
  }
}

export function revokeSid(sid: string) {
  if (sid) {
    revoked.add(sid);
  }
}

export function isRevoked(sid: string | undefined): boolean {
  return Boolean(sid) && revoked.has(sid as string);
}
