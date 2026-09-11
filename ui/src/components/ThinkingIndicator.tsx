// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { NuiLogo } from '@/components/NuiLogo'

interface Props {
  label?: string
  variant?: 'waiting' | 'streaming'
}

export function ThinkingIndicator({
  label,
  variant = 'waiting',
}: Props) {
  const resolvedLabel =
    label ?? (variant === 'streaming' ? 'Generating' : 'Agent is responding')

  return (
    <span
      className={`agui-thinking agui-thinking--${variant}`}
      role="status"
      aria-live="polite"
      aria-label={resolvedLabel}
    >
      {variant === 'streaming' ? (
        <span className="agui-thinking__logo-wrap" aria-hidden>
          <NuiLogo className="agui-thinking__logo" decorative />
          <NuiLogo className="agui-thinking__logo agui-thinking__shine" decorative />
        </span>
      ) : (
        <span className="agui-thinking__dots" aria-hidden>
          <span className="agui-thinking__dot" />
          <span className="agui-thinking__dot" />
          <span className="agui-thinking__dot" />
        </span>
      )}
      {variant === 'streaming' && (
        <span className="agui-thinking__label">{resolvedLabel}</span>
      )}
    </span>
  )
}
