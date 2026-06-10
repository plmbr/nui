// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { FolderOpen, Bot, Calendar } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import type { Project } from '@/types'

function formatAgentType(id: string) {
  return id.split('-').map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')
}

interface Props {
  project: Project
}

function Field({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="flex items-start gap-3">
      <div className="mt-0.5 text-muted-foreground">{icon}</div>
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm font-medium break-all">{value}</p>
      </div>
    </div>
  )
}

export function ProjectDetails({ project }: Props) {
  const created = new Date(project.createdAt).toLocaleString()

  return (
    <div className="flex flex-col gap-6 p-6 max-w-xl">
      <div>
        <h2 className="text-xl font-semibold">{project.name}</h2>
        <p className="text-sm text-muted-foreground mt-1">Project details</p>
      </div>
      <Separator />
      <div className="flex flex-col gap-5">
        <Field
          icon={<FolderOpen className="size-4" />}
          label="Working Directory"
          value={project.workingDir}
        />
        <Field
          icon={<Bot className="size-4" />}
          label="Agent Type"
          value={formatAgentType(project.agentType)}
        />
        <Field
          icon={<Calendar className="size-4" />}
          label="Created"
          value={created}
        />
      </div>
    </div>
  )
}
