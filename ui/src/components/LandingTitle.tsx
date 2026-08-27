// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useEffect, useState } from 'react'
import { NuiLogo } from '@/components/NuiLogo'
import { cn } from '@/lib/utils'

/** Prefix words before the nui logo; idioms that end in “big” (Hawaiian nui = great in size). */
const SLOGANS: readonly (readonly string[])[] = [
  ['tiny', 'but'],
  ['think'],
  ['dream'],
  ['win'],
  ['go'],
  ['make it'],
  ['start'],
  ['grow'],
  ['build'],
]

const WORD_MS = 1200

type Phase = number | 'logo'

function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined'
    && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function pickSlogan(): readonly string[] {
  return SLOGANS[Math.floor(Math.random() * SLOGANS.length)]!
}

export function LandingTitle() {
  const [words] = useState(pickSlogan)
  const [phase, setPhase] = useState<Phase>(() => (prefersReducedMotion() ? 'logo' : 0))

  useEffect(() => {
    if (prefersReducedMotion()) return

    const timers: number[] = []
    for (let i = 1; i < words.length; i++) {
      timers.push(window.setTimeout(() => setPhase(i), i * WORD_MS))
    }
    timers.push(window.setTimeout(() => setPhase('logo'), words.length * WORD_MS))

    return () => {
      for (const id of timers) window.clearTimeout(id)
    }
  }, [words])

  const showLogo = phase === 'logo'
  const currentWord = typeof phase === 'number' ? words[phase] : undefined

  return (
    <h1 className="landing-page__title" aria-label={showLogo ? undefined : currentWord}>
      <div className={cn('landing-page__title-stage', showLogo && 'landing-page__title-stage--logo')}>
        <span className="landing-page__title-words" aria-hidden={showLogo}>
          {currentWord != null && (
            <span
              key={`${phase}-${currentWord}`}
              className="landing-page__title-word landing-page__title-word--animate"
            >
              {currentWord}
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
