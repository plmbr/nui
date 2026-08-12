// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/contexts/theme'

export function ThemeSwitch() {
  const { theme, setTheme } = useTheme()
  const nextTheme = theme === 'light' ? 'dark' : 'light'

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      aria-label={`Switch to ${nextTheme} theme`}
      title={`Switch to ${nextTheme} theme`}
      className="app-header__theme-switch"
      onClick={() => setTheme(nextTheme)}
    >
      {theme === 'light' ? <Moon className="size-4.5" /> : <Sun className="size-4.5" />}
    </Button>
  )
}
