// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Plus, Settings2 } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface Props {
  onNewSession: () => void
  onCustomize: () => void
}

export function LandingPage({ onNewSession, onCustomize }: Props) {
  return (
    <div className="landing-page">
      <div className="landing-page__content">
        <h1 className="landing-page__title">The Loop</h1>
        <p className="landing-page__subtitle">
          Run agents in continuous sessions — pick up where you left off.
        </p>
        <div className="landing-page__actions">
          <Button size="lg" className="landing-page__btn-primary gap-2 px-6" onClick={onNewSession}>
            <Plus className="size-4" />
            New Session
          </Button>
          <Button size="lg" variant="outline" className="gap-2 px-6" onClick={onCustomize}>
            <Settings2 className="size-4" />
            Customize
          </Button>
        </div>
      </div>
    </div>
  )
}
