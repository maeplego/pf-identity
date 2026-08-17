const revoked = new Set<string>();
const seenJtis = new Set<string>();

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

export function consumeJti(jti: string): boolean {
  if (!jti || seenJtis.has(jti)) {
    return false;
  }
  seenJtis.add(jti);
  return true;
}
