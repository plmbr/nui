// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/contexts/theme'

export function ThemeSwitch() {
  const { theme, setTheme, canToggleMode, supportsDark, supportsLight, uiThemeDef } = useTheme()
  const nextTheme = theme === 'light' ? 'dark' : 'light'

  let title: string
  let ariaLabel: string
  if (!canToggleMode) {
    if (!supportsDark) {
      title = `${uiThemeDef.label} is light only`
      ariaLabel = title
    } else if (!supportsLight) {
      title = `${uiThemeDef.label} is dark only`
      ariaLabel = title
    } else {
      title = 'Color mode unavailable'
      ariaLabel = title
    }
  } else {
    title = `Switch to ${nextTheme} mode`
    ariaLabel = title
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      aria-label={ariaLabel}
      title={title}
      disabled={!canToggleMode}
      className="app-header__theme-switch"
      onClick={() => setTheme(nextTheme)}
    >
      {theme === 'light' ? <Moon className="size-4.5" /> : <Sun className="size-4.5" />}
    </Button>
  )
}
