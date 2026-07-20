// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { cn } from '@/lib/utils'
import type { AgentType } from '@/types'
import claudeCodeIcon from '@/assets/harness/claude-code.svg?url'
import piLightIcon from '@/assets/harness/pi-light.svg?url'
import piDarkIcon from '@/assets/harness/pi-dark.svg?url'
import codexIcon from '@/assets/harness/codex.svg?url'
import opencodeLightIcon from '@/assets/harness/opencode-light.svg?url'
import opencodeDarkIcon from '@/assets/harness/opencode-dark.svg?url'
import openaiIcon from '@/assets/harness/openai.svg?url'
import geminiIcon from '@/assets/harness/gemini.svg?url'
import openrouterIcon from '@/assets/harness/openrouter.svg?url'
import ollamaLightIcon from '@/assets/harness/ollama-light.svg?url'
import ollamaDarkIcon from '@/assets/harness/ollama-dark.svg?url'

type Harness = AgentType['harness']
type APIProvider = NonNullable<AgentType['provider']>
type IconKey = Harness | APIProvider

const HARNESS_ACCENTS: Partial<Record<IconKey, string>> = {
  'claude-code': '#d97757',
  anthropic: '#d97757',
  pi: '#8b5cf6',
  codex: '#10a37f',
  openai: '#10a37f',
  opencode: '#3b82f6',
  gemini: '#4285f4',
  openrouter: '#94A3B8',
  ollama: '#1f2937',
  docker: '#2496ed',
  devcontainer: '#2496ed',
  remote: '#64748b',
  extension: '#a855f7',
}

const BRAND_ICON_SRC: Partial<Record<IconKey, string | { light: string; dark: string }>> = {
  'claude-code': claudeCodeIcon,
  anthropic: claudeCodeIcon,
  pi: { light: piLightIcon, dark: piDarkIcon },
  codex: codexIcon,
  openai: openaiIcon,
  opencode: { light: opencodeLightIcon, dark: opencodeDarkIcon },
  gemini: geminiIcon,
  openrouter: openrouterIcon,
  ollama: { light: ollamaLightIcon, dark: ollamaDarkIcon },
}

interface Props {
  harness: Harness
  provider?: AgentType['provider']
  agentId?: string
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

function resolveIconKey(harness: Harness, provider?: AgentType['provider'], agentId?: string): IconKey {
  if (harness === 'api') {
    const apiProvider = provider?.trim() || agentId?.trim()
    if (apiProvider && apiProvider in BRAND_ICON_SRC) {
      return apiProvider as APIProvider
    }
  }
  return harness
}

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
    case 'api':
      return (
        <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden>
          <path
            stroke="currentColor"
            strokeWidth="1.75"
            strokeLinecap="round"
            d="M7 8h10M7 12h10M7 16h6"
          />
          <rect x="4" y="5" width="16" height="14" rx="2" stroke="currentColor" strokeWidth="1.75" />
        </svg>
      )
    case 'extension':
      return (
        <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden>
          <path
            stroke="currentColor"
            strokeWidth="1.75"
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M12 3v4M12 17v4M3 12h4M17 12h4M5.6 5.6l2.8 2.8M15.6 15.6l2.8 2.8M18.4 5.6l-2.8 2.8M8.4 15.6l-2.8 2.8"
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

export function HarnessIcon({ harness, provider, agentId, className, size = 'md' }: Props) {
  const iconKey = resolveIconKey(harness, provider, agentId)
  const brandSrc = BRAND_ICON_SRC[iconKey]
  const accent = HARNESS_ACCENTS[iconKey] ?? 'var(--muted-foreground)'

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
