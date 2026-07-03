// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

export const DEFAULT_SIDEBAR_WIDTH = 300
export const MIN_SIDEBAR_WIDTH = 200
export const MAX_SIDEBAR_WIDTH = 480

export function clampSidebarWidth(width: number): number {
  return Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, Math.round(width)))
}

export function resolveSidebarWidth(width?: number): number {
  if (width == null || !Number.isFinite(width)) {
    return DEFAULT_SIDEBAR_WIDTH
  }
  return clampSidebarWidth(width)
}
