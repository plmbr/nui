// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App'
import { isEmbedHost } from '@/lib/embedHost'
import { initEmbedClipboardHandlers } from '@/lib/embedClipboard'
import { initEmbedHostClipboardResultListener } from '@/lib/embedHostMessaging'
import { initExternalThemeListener } from '@/lib/externalTheme'

if (window.__NUI_DESKTOP__) {
  document.documentElement.classList.add('nui-desktop')
}

if (isEmbedHost()) {
  document.documentElement.dataset.embedHost = 'vscode'
}

initExternalThemeListener()
initEmbedHostClipboardResultListener()
initEmbedClipboardHandlers()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
