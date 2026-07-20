// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useMemo } from 'react'
import { PlumeriaFlower } from '@/components/PlumeriaFlower'
import { generateEvenRandomBlooms } from '@/lib/plumeriaBlooms'
import { cn } from '@/lib/utils'

export type PlumeriaOpacityVariant = 'default' | 'chat' | 'landing'

interface Props {
  count?: number
  avoidCenter?: boolean
  opacityVariant?: PlumeriaOpacityVariant
  /** Change to force a fresh random layout (e.g. panel open count, session id). */
  layoutKey?: string | number
  className?: string
}

/** Evenly scattered blooms with a fresh random layout each time layoutKey changes. */
export function PlumeriaRandomBackdrop({
  count = 5,
  avoidCenter = true,
  opacityVariant = 'default',
  layoutKey,
  className,
}: Props) {
  const blooms = useMemo(
    () => generateEvenRandomBlooms(count, { avoidCenter }),
    [count, avoidCenter, layoutKey],
  )

  return (
    <div
      className={cn(
        'plumeria-backdrop plumeria-backdrop--scattered',
        `plumeria-backdrop--${opacityVariant}`,
        className,
      )}
      aria-hidden="true"
    >
      {blooms.map((bloom) => (
        <div
          key={bloom.id}
          className="plumeria-backdrop__scatter"
          style={{
            left: `${bloom.left}%`,
            top: `${bloom.top}%`,
            transform: `translate(-50%, -50%) rotate(${bloom.initialRotation}deg)`,
          }}
        >
          <div
            className={cn(
              'plumeria-backdrop__bloom',
              'plumeria-backdrop__spin',
              `plumeria-backdrop__spin--${bloom.direction}`,
            )}
            style={{
              animationDuration: `${bloom.duration}s`,
              animationDelay: `${bloom.delay}s`,
            }}
          >
            <PlumeriaFlower size={bloom.size} />
          </div>
        </div>
      ))}
    </div>
  )
}

/** @deprecated Use PlumeriaRandomBackdrop */
export function PlumeriaBackdrop({ variant, className }: { variant: 'app' | 'chat'; className?: string }) {
  return (
    <PlumeriaRandomBackdrop
      count={variant === 'chat' ? 4 : 3}
      opacityVariant={variant === 'chat' ? 'chat' : 'default'}
      className={className}
    />
  )
}

/** @deprecated Use PlumeriaRandomBackdrop */
export function PlumeriaAmbient() {
  return <PlumeriaRandomBackdrop count={3} />
}
