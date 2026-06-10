// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { createContext, useContext, useEffect, useState } from 'react'
import { api } from '@/api'

type Theme = 'light' | 'dark'

interface ThemeContextValue {
  theme: Theme
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: 'light',
  setTheme: () => {},
})

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  // Initialize from localStorage for instant first paint — no flash on load
  const [theme, setThemeState] = useState<Theme>(() => {
    return (localStorage.getItem('theme') as Theme) ?? 'light'
  })

  // Apply theme class to DOM and keep localStorage in sync as fast-path cache
  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    localStorage.setItem('theme', theme)
  }, [theme])

  // On mount: reconcile with server (source of truth)
  useEffect(() => {
    api.settings.get()
      .then(s => { if (s.theme) setThemeState(s.theme) })
      .catch(() => {})
  }, [])

  function setTheme(t: Theme) {
    setThemeState(t)
    api.settings.update({ theme: t }).catch(() => {})
  }

  return <ThemeContext.Provider value={{ theme, setTheme }}>{children}</ThemeContext.Provider>
}

export function useTheme() {
  return useContext(ThemeContext)
}
