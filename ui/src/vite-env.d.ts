// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

/// <reference types="vite/client" />

declare const __NUI_VERSION__: string

interface Window {
  __NUI_DESKTOP__?: boolean
  __NUI_API_BASE__?: string
}
