// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { NuiLogo } from '@/components/NuiLogo'
import { cn } from '@/lib/utils'

type Phase = 'tiny' | 'but' | 'logo'

function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined'
    && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export function LandingTitle() {
  const [phase, setPhase] = useState<Phase>(() => (prefersReducedMotion() ? 'logo' : 'tiny'))

  useEffect(() => {
    if (prefersReducedMotion()) return

    const showBut = window.setTimeout(() => setPhase('but'), 800)
    const showLogo = window.setTimeout(() => setPhase('logo'), 1800)

    return () => {
      window.clearTimeout(showBut)
      window.clearTimeout(showLogo)
    }
  }, [])

  const showLogo = phase === 'logo'
  const ariaLabel = phase === 'but' ? 'but' : 'tiny'

  return (
    <h1 className="landing-page__title" aria-label={showLogo ? undefined : ariaLabel}>
      <div className={cn('landing-page__title-stage', showLogo && 'landing-page__title-stage--logo')}>
        <span className="landing-page__title-words" aria-hidden={showLogo}>
          {phase === 'tiny' && (
            <span className="landing-page__title-word landing-page__title-word--animate">
              tiny
            </span>
          )}
          {phase === 'but' && (
            <span className="landing-page__title-word landing-page__title-word--animate">
              but
            </span>
          )}
        </span>
        <span className="landing-page__title-logo" aria-hidden={!showLogo}>
          <NuiLogo decorative={!showLogo} />
        </span>
      </div>
    </h1>
  )
}
