# Loop Sandboxing Options — Research Report

> Sources: SandboxEscapeBench (Oxford/UK AISI, March 2026), go-sandbox, bubblewrap, nsjail, gVisor docs.

## Summary

Loop needs a platform-split sandboxing strategy. The Linux path has multiple solid options. The macOS path is limited and officially deprecated. Docker alone is provably insufficient against capable LLM agents.

The current `ClaudeCodeAgent` path (direct subprocess in `internal/agent/claude_code.go`) has no sandboxing. The Docker extension agent path is better but still escapable if misconfigured.

---

## Linux options (ranked)

### 1. go-sandbox — recommended

A Go-native library combining namespace isolation, seccomp-bpf, ptrace, rlimits, and cgroups in a pre-forked container model.

**Latency benchmarks:**
- ~2ms — pre-forked container reuse (BenchmarkContainer)
- ~14ms — full unshare per invocation (BenchmarkUnshareAll)
- ~24ms — unshare + pivot_root (BenchmarkUnshareMountPivot)

**Isolation layers:** mount, PID, user, UTS, cgroup, IPC, network namespaces + `SECCOMP_RET_KILL_PROCESS` (kernel ≥4.14) + ptrace + rlimits + cgroups v1/v2.

**Integration:** The `Runner` interface (`Run(context.Context) Result`) maps onto Loop's `Agent` interface. Replace the `exec.Command("claude", ...)` call in `ClaudeCodeAgent.Run()` with an `Environment.Execve()` call.

Source: https://github.com/criyle/go-sandbox

---

### 2. bubblewrap (bwrap) via os/exec

Construct bwrap arguments programmatically, append `-- claude ...` at the end.

```go
args := []string{
    "--unshare-all",
    "--ro-bind", "/usr", "/usr",
    "--ro-bind", "/lib", "/lib",
    "--bind", workDir, workDir,
    "--chdir", workDir,
    "--proc", "/proc",
    "--dev", "/dev",
    "--tmpfs", "/tmp",
    "--",
    "claude", "-p", message, "--output-format", "stream-json",
    "--verbose", "--dangerously-skip-permissions",
}
cmd := exec.Command("bwrap", args...)
```

A Go wrapper package exists: `github.com/0xmhha/cli-wrapper/pkg/sandbox/providers/bubblewrap` (returns `ErrUnsupportedPlatform` on non-Linux — good for build tag split).

**Namespaces used:** CLONE_NEWUSER, CLONE_NEWIPC, CLONE_NEWPID, CLONE_NEWNET, CLONE_NEWUTS, CLONE_NEWNS (mount, always on).

**Caveat:** bwrap setuid mode has NOT been fully removed — privilege requirements depend on distro and `kernel.unprivileged_userns_clone` sysctl. Test on target distro before assuming unprivileged operation.

Source: https://github.com/containers/bubblewrap

---

### 3. nsjail via os/exec

Production-proven (used by Chromium). Invoked as a subprocess wrapper with `--` separator.

```go
args := []string{
    "-Mo",
    "--chroot", "/",
    "--user", "99999",
    "--seccomp_string", "ALLOW { read, write, exit, exit_group } DEFAULT KILL",
    "--", "claude", "-p", message, "--output-format", "stream-json",
    "--verbose", "--dangerously-skip-permissions",
}
cmd := exec.Command("nsjail", args...)
```

Kafel DSL (`-P policy_file` or `--seccomp_string`) allows named syscall allowlists. More configurable than bwrap for seccomp, but heavier external dependency.

Sources: https://github.com/google/nsjail, https://pkg.go.dev/go.chromium.org/infra/isolation/nsjail_wrapper

---

### 4. Native SysProcAttr.Cloneflags — zero external deps

Add namespace isolation directly in `ClaudeCodeAgent.Run()` with no external binary:

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWNS |
                syscall.CLONE_NEWUTS |
                syscall.CLONE_NEWIPC |
                syscall.CLONE_NEWPID |
                syscall.CLONE_NEWNET |
                syscall.CLONE_NEWUSER,
}
```

Caveat: `CLONE_NEWNS` cannot be combined with `CLONE_FS`. Gives namespace isolation but no seccomp filtering or filesystem bind-mounts without additional setup.

---

### 5. gVisor / Firecracker — stronger but heavier

- **gVisor**: intercepts syscalls in user-space (Sentry process) before reaching host kernel. OCI-compatible — can wrap the existing Docker agent path. Not a full hardware boundary.
- **Firecracker microVMs**: hardware-enforced boundary. Strongest isolation available. High integration complexity — not practical for per-project subprocess wrapping without significant rearchitecture. Relevant only if Loop becomes a multi-tenant hosted service.

---

## macOS

**Only option: `sandbox-exec` (Seatbelt) + dedicated limited user account.**

```sh
sandbox-exec -f agent.sb -D WORK_DIR=/path/to/project claude -p ...
```

Sample profile:
```scheme
(version 1)
(deny default)
(allow file-read* (subpath (param "WORK_DIR")))
(allow file-write* (subpath (param "WORK_DIR")))
(allow process-exec)
(allow signal)
```

**Critical limitations:**
- Officially **deprecated by Apple** — long-term maintenance risk.
- macOS does **not** support nested sandboxes. If Loop's Go server itself is sandboxed, child processes cannot also apply `sandbox-exec` (returns `sandbox_apply_container: Operation not permitted`).
- Network blocking via Seatbelt does **NOT** reliably intercept `connect()` syscalls to non-localhost IPs at the kernel level (refuted 0-3). Network egress requires supplementary measures (e.g., `pf` rules).

Use the limited-user-account layer as defense-in-depth on macOS, but treat Linux as the primary platform for real sandboxing.

---

## Docker (existing approach)

Plain Docker is **insufficient** against capable LLM agents (SandboxEscapeBench, Oxford/UK AISI, March 2026). Frontier models reliably escape difficulty 1–2 Docker scenarios: `--privileged` containers, writable host mounts, exposed Docker sockets. Well-configured Docker is better but lacks a hardware boundary.

Loop's current Docker mode (`docker run -d -p 127.0.0.1::containerPort`) is reasonable for extension agents. The `ClaudeCodeAgent` path (direct subprocess) has **no sandboxing at all** today.

---

## Integration plan

| Platform | Approach | Effort |
|---|---|---|
| Linux | `go-sandbox` wrapping `ClaudeCodeAgent.Run()` | Medium — adapt Runner interface |
| Linux (simpler) | `bwrap` args via `os/exec` + build tag | Low — just arg construction |
| Linux (no deps) | `SysProcAttr.Cloneflags` inline | Low — no seccomp/fs restriction |
| macOS | `sandbox-exec` profile + limited user | Medium — profile authoring |
| Both | Keep Docker for extension/remote agents | Already done |

---

## Open questions before implementing

1. Does `claude` CLI need loopback access (for MCP servers on localhost)? Determines whether `--unshare-net` is usable or needs a loopback-only network namespace.
2. Does Loop's production Linux server run as root or non-root? Is `kernel.unprivileged_userns_clone` enabled? Determines whether bwrap/nsjail need setuid wrappers.
3. Is there a Go-native Landlock LSM binding for kernel-level filesystem access control (complement to namespaces, alternative to bwrap bind-mounts)?
4. Apple's roadmap for replacing `sandbox-exec` — is there an endorsed API for non-app-bundle process launchers before it's removed?
