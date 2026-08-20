export function issuer(): string {
  const v = process.env.OIDC_ISSUER?.replace(/\/$/, "");
  if (!v) {
    throw new Error("OIDC_ISSUER is required");
  }
  return v;
}

// Browser redirects use issuer(); server-side fetch uses this so Compose can reach the IdP by service name.
export function internalBase(): string {
  const v = process.env.OIDC_INTERNAL_BASE?.replace(/\/$/, "");
  return v || issuer();
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

export function postLogoutRedirectUri(): string {
  const v = process.env.OIDC_POST_LOGOUT_REDIRECT_URI;
  if (v) {
    return v;
  }
  return redirectUri().replace(/\/callback$/, "/logged-out");
}

export function rpLabel(): string {
  const v = process.env.OIDC_RP_LABEL?.trim();
  if (v) {
    return v;
  }
  return clientId();
}

// Browser-facing origin. Next.js in Compose listens on 0.0.0.0 so req.nextUrl.origin is unusable.
export function publicOrigin(req: { headers: Headers; nextUrl: URL }): string {
  const redirect = process.env.OIDC_REDIRECT_URI?.trim();
  if (redirect) {
    try {
      return new URL(redirect).origin;
    } catch {
      // fall through
    }
  }
  const host = req.headers.get("x-forwarded-host") || req.headers.get("host");
  if (host && !host.startsWith("0.0.0.0")) {
    const proto = req.headers.get("x-forwarded-proto") || "http";
    return `${proto.split(",")[0].trim()}://${host.split(",")[0].trim()}`;
  }
  return "http://localhost:3001";
}
