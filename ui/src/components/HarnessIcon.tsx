// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { cn } from '@/lib/utils'
import type { AgentType } from '@/types'
import claudeCodeIcon from '@/assets/harness/claude-code.svg?url'
import piLightIcon from '@/assets/harness/pi-light.svg?url'
import piDarkIcon from '@/assets/harness/pi-dark.svg?url'
import codexIcon from '@/assets/harness/codex.svg?url'
import opencodeLightIcon from '@/assets/harness/opencode-light.svg?url'
import opencodeDarkIcon from '@/assets/harness/opencode-dark.svg?url'

type Harness = AgentType['harness']

const HARNESS_ACCENTS: Partial<Record<Harness, string>> = {
  'claude-code': '#d97757',
  pi: '#8b5cf6',
  codex: '#10a37f',
  opencode: '#3b82f6',
  docker: '#2496ed',
  devcontainer: '#2496ed',
  remote: '#64748b',
}

const BRAND_ICON_SRC: Partial<Record<Harness, string | { light: string; dark: string }>> = {
  'claude-code': claudeCodeIcon,
  pi: { light: piLightIcon, dark: piDarkIcon },
  codex: codexIcon,
  opencode: { light: opencodeLightIcon, dark: opencodeDarkIcon },
}

interface Props {
  harness: Harness
  className?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
}

const SIZE = {
  sm: 'size-7',
  md: 'size-9',
  lg: 'size-12',
  xl: 'size-16',
} as const

const IMG_SIZE = {
  sm: 'size-4',
  md: 'size-5',
  lg: 'size-8',
  xl: 'size-11',
} as const

function HarnessGlyph({ harness, className }: { harness: Harness; className?: string }) {
  switch (harness) {
    case 'docker':
    case 'devcontainer':
      return (
        <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden>
          <rect x="3" y="10" width="3.5" height="3.5" fill="currentColor" />
          <rect x="7" y="10" width="3.5" height="3.5" fill="currentColor" />
          <rect x="11" y="10" width="3.5" height="3.5" fill="currentColor" />
          <rect x="7" y="6.5" width="3.5" height="3.5" fill="currentColor" />
          <path
            stroke="currentColor"
            strokeWidth="1.75"
            strokeLinecap="round"
            d="M15.5 13.5h2.5c1.2 0 2.2-1 2.2-2.2 0-1-.7-1.8-1.6-2.1"
          />
        </svg>
      )
    case 'remote':
      return (
        <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden>
          <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.75" />
          <path
            stroke="currentColor"
            strokeWidth="1.75"
            strokeLinecap="round"
            d="M12 4v2M12 18v2M4 12h2M18 12h2M6.3 6.3l1.4 1.4M16.3 16.3l1.4 1.4M17.7 6.3l-1.4 1.4M7.7 16.3l-1.4 1.4"
          />
        </svg>
      )
    default:
      return (
        <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden>
          <circle cx="12" cy="12" r="4" fill="currentColor" />
        </svg>
      )
  }
}

function BrandIcon({
  src,
  className,
}: {
  src: string | { light: string; dark: string }
  className?: string
}) {
  if (typeof src === 'string') {
    return <img src={src} alt="" className={cn('object-contain', className)} draggable={false} />
  }

  return (
    <>
      <img src={src.light} alt="" className={cn('object-contain dark:hidden', className)} draggable={false} />
      <img src={src.dark} alt="" className={cn('hidden object-contain dark:block', className)} draggable={false} />
    </>
  )
}

export function HarnessIcon({ harness, className, size = 'md' }: Props) {
  const brandSrc = BRAND_ICON_SRC[harness]
  const accent = HARNESS_ACCENTS[harness] ?? 'var(--muted-foreground)'

  if (brandSrc) {
    return (
      <span
        className={cn(
          'inline-flex shrink-0 items-center justify-center rounded-lg bg-muted/35',
          SIZE[size],
          className,
        )}
      >
        <BrandIcon src={brandSrc} className={IMG_SIZE[size]} />
      </span>
    )
  }

  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center justify-center rounded-lg',
        SIZE[size],
        className,
      )}
      style={{
        backgroundColor: `color-mix(in oklch, ${accent} 18%, transparent)`,
        color: accent,
      }}
    >
      <HarnessGlyph harness={harness} className={IMG_SIZE[size]} />
    </span>
  )
}
