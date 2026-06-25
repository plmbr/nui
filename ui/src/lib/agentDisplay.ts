// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export function harnessLabel(harness: string, sandbox?: string): string {
  if (sandbox === 'docker') return `${harness} · docker`
  if (sandbox === 'bubblewrap') return `${harness} · bwrap`
  return harness
}
