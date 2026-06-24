// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import { useCallback, useEffect, useState } from 'react'
import { Trash2 } from 'lucide-react'
import { ConfirmDeleteDialog } from '@/components/ConfirmDeleteDialog'
import { Button } from '@/components/ui/button'
import { api } from '@/api'
import type { SkillEntry } from '@/types'

export function SkillsTab() {
  const [skills, setSkills] = useState<SkillEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [removing, setRemoving] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const list = await api.skills.list()
      setSkills(list.sort((a, b) => a.name.localeCompare(b.name)))
    } catch {
      setSkills([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const remove = async (name: string) => {
    setRemoving(name)
    try {
      await api.skills.remove(name)
      await load()
    } finally {
      setRemoving(null)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading skills…</p>
  }

  return (
    <div className="customize-tab-content space-y-4">
      <p className="text-sm text-muted-foreground">
        Skills installed in <code className="text-xs">~/.loop/skills/</code>. Reference them in agent
        definitions with <code className="text-xs">ref: my-skill</code>.
      </p>

      {skills.length === 0 ? (
        <p className="text-sm text-muted-foreground">No skills installed yet.</p>
      ) : (
        <ul className="divide-y rounded-lg border">
          {skills.map((skill) => (
            <li key={skill.name} className="flex items-start justify-between gap-4 p-4">
              <div className="min-w-0">
                <p className="font-medium text-sm">{skill.name}</p>
                <p className="text-xs text-muted-foreground mt-1">
                  Source: {skill.source}
                  {skill.path && ` · ${skill.path}`}
                  {skill.git && ` · ${skill.git}`}
                </p>
                {skill.installedAt && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Installed {new Date(skill.installedAt).toLocaleString()}
                  </p>
                )}
              </div>
              <Button
                variant="ghost"
                size="sm"
                disabled={removing === skill.name}
                onClick={() => setDeleteTarget(skill.name)}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </li>
          ))}
        </ul>
      )}

      <ConfirmDeleteDialog
        open={deleteTarget != null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}
        title="Remove skill?"
        description={
          <>
            This will permanently remove the skill <strong>{deleteTarget}</strong> from your Loop
            installation. This action cannot be undone.
          </>
        }
        confirmLabel="Remove"
        confirming={deleteTarget != null && removing === deleteTarget}
        onConfirm={async () => {
          if (deleteTarget) await remove(deleteTarget)
        }}
      />
    </div>
  )
}
