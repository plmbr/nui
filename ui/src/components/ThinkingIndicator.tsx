// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

interface Props {
  label?: string
}

export function ThinkingIndicator({ label = 'Agent is responding' }: Props) {
  return (
    <span className="agui-thinking" role="status" aria-live="polite" aria-label={label}>
      <span className="agui-thinking__dot" />
      <span className="agui-thinking__dot" />
      <span className="agui-thinking__dot" />
    </span>
  )
}
