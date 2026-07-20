// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useId } from 'react'
import { cn } from '@/lib/utils'

interface Props {
  className?: string
  /** Visual size in pixels. */
  size?: number
}

/** Spiral offsets give the classic overlapping plumeria pinwheel. */
const PETALS = [
  { angle: -18, variant: 'pink' as const },
  { angle: 54, variant: 'gold' as const },
  { angle: 126, variant: 'pink' as const },
  { angle: 198, variant: 'gold' as const },
  { angle: 270, variant: 'pink' as const },
]

const PETAL_PATH =
  'M 24 22.6 C 19.4 21.2 17.6 14.2 19.2 7.4 C 20.5 3.6 24 2.2 27.5 3.6 C 29.1 7.4 28.6 14.2 24 22.6 Z'

const HIGHLIGHT_PATH =
  'M 24 21.2 C 21.2 20.2 20.2 14.8 21.4 9.2 C 22.2 6.2 24 5.2 25.8 6.2 C 27 9.2 26.2 14.8 24 21.2 Z'

/** Five-petal plumeria (pua melia) — waxy spiral bloom with island colors. */
export function PlumeriaFlower({ className, size = 48 }: Props) {
  const uid = useId().replace(/:/g, '')

  return (
    <svg
      viewBox="0 0 48 48"
      width={size}
      height={size}
      className={cn('plumeria-flower', className)}
      aria-hidden="true"
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        {PETALS.map(({ angle, variant }) => (
          <linearGradient
            key={`${angle}-${variant}`}
            id={`plumeria-petal-${uid}-${angle}`}
            x1="24"
            y1="22"
            x2="24"
            y2="2.5"
            gradientUnits="userSpaceOnUse"
            gradientTransform={`rotate(${angle} 24 24)`}
          >
            <stop offset="0%" stopColor="var(--plumeria-throat)" />
            <stop offset="32%" stopColor="var(--plumeria-petal-base)" />
            <stop offset="72%" stopColor="var(--plumeria-petal-mid)" />
            <stop
              offset="100%"
              stopColor={
                variant === 'pink' ? 'var(--plumeria-petal-tip)' : 'var(--plumeria-petal-gold)'
              }
            />
          </linearGradient>
        ))}
        <radialGradient id={`plumeria-center-${uid}`} cx="50%" cy="48%" r="42%">
          <stop offset="0%" stopColor="var(--plumeria-center-bright)" />
          <stop offset="55%" stopColor="var(--plumeria-center)" />
          <stop offset="100%" stopColor="var(--plumeria-throat)" />
        </radialGradient>
        <filter id={`plumeria-shadow-${uid}`} x="-30%" y="-30%" width="160%" height="160%">
          <feDropShadow dx="0" dy="0.6" stdDeviation="0.7" floodColor="var(--plumeria-shadow)" />
        </filter>
      </defs>
      <g filter={`url(#plumeria-shadow-${uid})`}>
        {PETALS.map(({ angle }) => (
          <g key={angle} transform={`rotate(${angle} 24 24)`}>
            <path
              d={PETAL_PATH}
              fill={`url(#plumeria-petal-${uid}-${angle})`}
              stroke="var(--plumeria-edge)"
              strokeWidth="0.28"
              strokeLinejoin="round"
            />
            <path d={HIGHLIGHT_PATH} fill="var(--plumeria-petal-highlight)" opacity="0.42" />
          </g>
        ))}
        <circle cx="24" cy="24" r="5.8" fill={`url(#plumeria-center-${uid})`} />
        <circle cx="24" cy="24" r="3.4" fill="var(--plumeria-center)" opacity="0.9" />
        <circle cx="23.2" cy="22.8" r="1.5" fill="var(--plumeria-center-bright)" opacity="0.95" />
      </g>
    </svg>
  )
}
