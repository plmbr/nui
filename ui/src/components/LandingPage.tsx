// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { Plus, Settings } from 'lucide-react'
import { NuiLogo } from '@/components/NuiLogo'
import { Button } from '@/components/ui/button'

interface Props {
  onNewSession: () => void
  onCustomize: () => void
}

export function LandingPage({ onNewSession, onCustomize }: Props) {
  return (
    <div className="landing-page">
      <div className="landing-page__content">
        <h1 className="landing-page__title" aria-label="nui">
          <NuiLogo decorative />
        </h1>
        <p className="landing-page__slogan">
          tiny but <span>nui</span>
        </p>
   
        <p className="landing-page__subtitle">
          Scale any task with AI — scale your work, scale your life.
        </p>
        <div className="landing-page__actions">
          <Button size="lg" className="landing-page__btn-primary gap-2 px-6" onClick={onNewSession}>
            <Plus className="size-4" />
            New Session
          </Button>
          <Button size="lg" variant="outline" className="gap-2 px-6" onClick={onCustomize}>
            <Settings className="size-4" />
            Customize
          </Button>
        </div>
      </div>
    </div>
  )
}
