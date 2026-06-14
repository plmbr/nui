# go-sandbox vs bubblewrap — Comparison Report

## Verdict

**Use bubblewrap. go-sandbox is not appropriate for this use case.**

go-sandbox is a niche competitive programming judge library with no documented production users outside the OJ ecosystem. Bubblewrap is what Anthropic itself uses to sandbox the claude CLI on Linux.

---

## go-sandbox — what it actually is

| Fact | Detail |
|---|---|
| Stars | 263 (GitHub confirmed) |
| Maintainer | Single individual (`criyle`) |
| Origin | Port of `uoj-judger/run_program` — a competitive programming judge |
| Companion | `go-judge` — explicitly tagged `oj` / `online-judge` on GitHub |
| Production users | None documented outside OJ systems |
| Org backing | None |
| pkg.go.dev importers | ~106 — none identified as large-scale non-OJ deployments |

The OJ use case is architecturally different from AI agent sandboxing: OJ judges run short-lived, predictable student programs. AI agents run long-lived, unpredictable, network-hungry workloads. The 2ms benchmark is optimized for OJ's high-frequency short-run pattern, not one-agent-process-per-project.

---

## bubblewrap — production pedigree

| Fact | Detail |
|---|---|
| Maintainer | `containers` org (Red Hat / GNOME / open-source coalition) |
| Ships in | Every major Linux distro |
| Used by | Flatpak, rpm-ostree, bwrap-oci, libgnome-desktop, sandwine |
| CVEs | 2 total — **both setuid-only**, not exploitable in unprivileged mode |
| Setuid status | Deprecated in 0.11.2, removed in 0.12.0 — unprivileged namespaces is the intended model |
| Security response | Active — CVE-2026-41163 patched same release cycle |

---

## The key reference: Anthropic's sandbox-runtime

Anthropic's own `sandbox-runtime` uses bubblewrap for Linux subprocess sandboxing of the claude CLI.

Their pattern (from `src/sandbox/linux-sandbox-utils.ts`):

```
bwrap --unshare-net [bind mounts] -- claude -p ...
```

The network namespace is removed entirely. All traffic is routed through Unix socket proxies on the host that enforce domain allowlists/denylists. This answers whether `--unshare-net` is safe: claude CLI's network access goes through a proxy, not directly.

Source: https://github.com/anthropic-experimental/sandbox-runtime

---

## CVE track record

| CVE | CVSS | Exploitable in unprivileged mode? |
|---|---|---|
| CVE-2020-5291 | 7.8 High | No — required setuid bwrap AND unprivileged namespaces simultaneously |
| CVE-2026-41163 | — | No — required setuid mode (deprecated, off by default since Debian 11 / RHEL 8) |

Modern distros ship bwrap without setuid. Neither CVE applies in the standard deployment model.

---

## Operational concern: Ubuntu 24.04+ AppArmor

Ubuntu 24.04+ restricts unprivileged user namespaces by default via AppArmor. Requires per-app allowlisting:

```sh
# option 1: complain mode (permissive, logs violations)
sudo aa-complain /usr/bin/bwrap

# option 2: write a specific AppArmor profile for the loop binary
```

This is a known, documented issue with a standard fix. Must be documented in Loop's install instructions for Ubuntu deployments.

---

## Comparison table

| Dimension | go-sandbox | bubblewrap |
|---|---|---|
| Maintenance | Single maintainer | `containers` org, major distros |
| Production users | OJ systems only | Flatpak, GNOME, Anthropic sandbox-runtime |
| Adoption signal | 263 stars | Ships in every distro |
| Security CVEs | None (too obscure) | 2 CVEs, both setuid-only, both fixed |
| Privilege model | Requires kernel namespaces | Unprivileged user namespaces (setuid deprecated) |
| Network isolation | Yes | Yes (`--unshare-net`) + proxy pattern |
| Seccomp | Yes | Yes (`--seccomp` FD, caller-compiled BPF) |
| Go integration | Native library | `os/exec` arg construction |
| AI agent reference | None | Anthropic sandbox-runtime (direct analogue) |
| Risk | High (no external validation) | Low (infrastructure-grade) |

---

## Recommended integration for Loop

Follow the Anthropic sandbox-runtime pattern in `internal/agent/claude_code.go`:

1. Construct `bwrap` args — `--unshare-all`, bind-mount project `workDir` RW, everything else RO or absent
2. Use `--unshare-net` + Unix socket proxy on the host for outbound traffic (if claude CLI needs internet access)
3. Add a compiled seccomp BPF filter via `--seccomp` FD for syscall allowlisting
4. On Ubuntu 24.04+: document AppArmor allowlisting in install instructions
5. Build-tag the bwrap path as Linux-only; keep unmodified subprocess path for macOS

---

## Note on previous sandboxing report

`docs/sandboxing-options.md` recommended go-sandbox as the top option. That recommendation is superseded by this report. go-sandbox should be removed from consideration. Bubblewrap is the correct choice.
