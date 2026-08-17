import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Token exchange stays on this origin so the IdP does not need CORS on /token.
};

export default nextConfig;
