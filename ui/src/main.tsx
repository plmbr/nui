// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App'

if (window.__NUI_DESKTOP__) {
  document.documentElement.classList.add('nui-desktop')
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
