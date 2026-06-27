// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

interface Props {
  label?: string
  variant?: 'waiting' | 'streaming'
}

export function ThinkingIndicator({
  label,
  variant = 'waiting',
}: Props) {
  const resolvedLabel =
    label ?? (variant === 'streaming' ? 'Generating…' : 'Agent is responding')

  return (
    <span
      className={`agui-thinking agui-thinking--${variant}`}
      role="status"
      aria-live="polite"
      aria-label={resolvedLabel}
    >
      <span className="agui-thinking__dots" aria-hidden>
        <span className="agui-thinking__dot" />
        <span className="agui-thinking__dot" />
        <span className="agui-thinking__dot" />
      </span>
      {variant === 'streaming' && (
        <span className="agui-thinking__label">{resolvedLabel}</span>
      )}
    </span>
  )
}
