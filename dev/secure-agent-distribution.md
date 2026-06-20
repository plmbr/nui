# Secure Agent Distribution

## Problem

Users want to publish their own agents (as zip files) and only their authorized users should be able to install and run them. The agent zip needs to be:
1. Verified as coming from the publisher (integrity + authenticity)
2. Only executable by authorized users (access control)

---

## Option 1: Signing with minisign (Public Agents)

### How it works

Publishers sign their agent zip with a private key. Loop verifies the signature before executing.

**Files distributed:**
- `agent.zip` — the agent bundle
- `agent.zip.minisig` — the detached signature

**Loop client side:**
- Contains (or is given) the publisher's public key
- Verifies `agent.zip.minisig` against `agent.zip` before running

### Why the public key being visible is not a problem

In asymmetric cryptography, security comes from the **private key being secret**, not the public key. The public key's job is to be distributed as widely as possible.

Even if an attacker knows the public key, they **cannot forge a signature** without the private key. That's the mathematical guarantee of Ed25519.

To publish a malicious zip that Loop accepts, an attacker would need to:
- Steal the publisher's private key (never shared, never in the repo)
- Or break Ed25519 cryptography (computationally infeasible)

**Real-world precedent:** apt/yum, Homebrew, minisign itself — all use this model with public keys visible in open source code.

### Actual risks

| Risk | Mitigation |
|---|---|
| Private key stolen | Keep it offline, use a hardware key (YubiKey) |
| Build pipeline compromised | Sign on a separate trusted machine |
| Attacker submits PR to change the trusted public key | Code review, signed commits, community scrutiny |

---

## Option 2: Encryption for Access Control (Private Agents)

### Use case

The publisher wants only specific users to be able to install and run the agent. Signing alone doesn't help here — it proves authenticity but anyone can still read/run the zip.

### How it works

**Publisher side:**
1. Encrypts `agent.zip` with a symmetric key (e.g., AES-256)
2. Distributes the encrypted `agent.zip.enc` publicly (e.g., on GitHub)
3. Shares the decryption key out-of-band with authorized users only

**User side:**
1. Sets the decryption key as an environment variable: `AGENT_KEY=<base64-key>`
2. Loop reads the env var, decrypts the zip in memory, then executes

### Combined: Sign + Encrypt

For maximum security, do both:
1. Publisher signs the plaintext zip → `agent.zip.minisig`
2. Publisher encrypts the zip → `agent.zip.enc`
3. User decrypts → `agent.zip` (in memory)
4. Loop verifies signature against decrypted content
5. Loop executes

This gives both **access control** (only key holders can decrypt) and **integrity** (signature proves it came from the publisher and wasn't tampered with after encryption).

### Key distribution options

| Method | Trade-offs |
|---|---|
| Environment variable (`AGENT_KEY`) | Simple, works well for CLI tools and CI |
| `.env` file (gitignored) | Convenient locally, easy to leak |
| Secrets manager (Vault, AWS SSM) | Best for teams, more infrastructure |
| Per-user asymmetric encryption | Each user has a keypair; publisher encrypts to each user's public key — most secure, most complex |

### Per-user asymmetric variant (most secure)

Each authorized user generates an Ed25519 or X25519 keypair. Publisher encrypts the agent key *to each user's public key*. Only the holder of the private key can decrypt.

- No shared secret — compromise of one user doesn't expose others
- Publisher can revoke access by simply not including a user's encrypted key in future releases
- Private key stays on the user's machine, never shared

### Environment variable approach (simplest)

```bash
# User sets this in their shell profile or CI secrets
export LOOP_AGENT_KEY="base64encodedkey..."

# Loop decrypts before running
loop install https://github.com/publisher/agent/releases/latest/agent.zip.enc
```

Loop workflow:
1. Download `agent.zip.enc`
2. Read `LOOP_AGENT_KEY` from env
3. Decrypt in memory → `agent.zip`
4. Optionally verify signature
5. Execute

---

## Recommendation

| Scenario | Approach |
|---|---|
| Open source agent, public distribution | minisign signing only |
| Private agent, small team | Signing + symmetric encryption via env var |
| Private agent, larger team or enterprise | Signing + per-user asymmetric encryption |

Start with the env var approach — it's simple to implement and well understood. Upgrade to per-user keys if the user base grows or if key rotation becomes a concern.
