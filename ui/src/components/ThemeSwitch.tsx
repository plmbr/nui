// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/contexts/theme'
import { cn } from '@/lib/utils'

export function ThemeSwitch() {
  const { theme, setTheme } = useTheme()

  return (
    <div className="app-header__theme-switch" role="group" aria-label="Theme">
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-pressed={theme === 'light'}
        aria-label="Light theme"
        className={cn(theme === 'light' && 'bg-background text-foreground shadow-sm')}
        onClick={() => setTheme('light')}
      >
        <Sun className="size-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-pressed={theme === 'dark'}
        aria-label="Dark theme"
        className={cn(theme === 'dark' && 'bg-background text-foreground shadow-sm')}
        onClick={() => setTheme('dark')}
      >
        <Moon className="size-4" />
      </Button>
    </div>
  )
}
