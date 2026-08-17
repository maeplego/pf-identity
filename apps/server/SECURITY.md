# Security policy

This repository is a **learning IdP**. Do not use it to protect real user data or production systems.

## Reporting

Do not file public issues that include secrets, private keys, or exploit proofs of concept. Describe the class of issue and a safe reproduction if needed.

## Repository rules

- Secrets live in environment variables. Never commit `.env`, PEM files, or cookies dumped from a browser.
- Authorization codes, refresh tokens, session tokens, and client secrets are stored hashed.
- Cryptographic primitives come from `golang.org/x/crypto` and `github.com/lestrrat-go/jwx`. Do not hand-roll signatures.
