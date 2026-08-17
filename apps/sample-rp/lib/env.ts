export function issuer(): string {
  const v = process.env.OIDC_ISSUER?.replace(/\/$/, "");
  if (!v) {
    throw new Error("OIDC_ISSUER is required");
  }
  return v;
}

export function clientId(): string {
  const v = process.env.OIDC_CLIENT_ID;
  if (!v) {
    throw new Error("OIDC_CLIENT_ID is required");
  }
  return v;
}

export function redirectUri(): string {
  const v = process.env.OIDC_REDIRECT_URI;
  if (!v) {
    throw new Error("OIDC_REDIRECT_URI is required");
  }
  return v;
}
