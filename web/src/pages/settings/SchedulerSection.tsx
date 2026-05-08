import { useCallback, useEffect, useState } from 'react'
import { Clock3, Loader2, Pencil, Play, RefreshCw, Power, PowerOff } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { scheduledTasksApi } from '@/api/scheduledTasks'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type {
  ScheduledTask,
  ScheduledTaskResult,
  ScheduledTaskRun,
  ScheduledTaskTriggerSource,
} from '@/types/scheduledTask'
import { cn } from '@/lib/utils'

type RunStatusFilter = 'all' | ScheduledTaskResult
type TriggerSourceFilter = 'all' | ScheduledTaskTriggerSource

type EditFormState = {
  cronSpec: string
  enabled: boolean
  runTimeoutSeconds: string
}

function formatDateTime(value: string | null, locale: string, fallback: string): string {
  if (!value) return fallback
  return new Date(value).toLocaleString(locale)
}

function formatDuration(durationMs: number): string {
  if (!durationMs) return '0 ms'
  if (durationMs < 1000) return `${durationMs} ms`
  return `${(durationMs / 1000).toFixed(durationMs >= 10_000 ? 0 : 1)} s`
}

function taskResultBadge(result: string, t: (key: string) => string) {
  switch (result) {
    case 'success':
      return <Badge variant="default">{t('taskResultSuccess')}</Badge>
    case 'failed':
      return <Badge variant="destructive">{t('taskResultFailed')}</Badge>
    case 'running':
      return <Badge variant="secondary">{t('taskResultRunning')}</Badge>
    case 'skipped':
      return <Badge variant="outline">{t('taskResultSkipped')}</Badge>
    default:
      return <span className="text-sm text-on-surface-variant">{t('taskResultNone')}</span>
  }
}

function taskStateBadge(task: ScheduledTask, t: (key: string) => string) {
  if (!task.registered) return <Badge variant="outline">{t('taskStateRetired')}</Badge>
  if (task.state === 'running') return <Badge variant="default">{t('taskStateRunning')}</Badge>
  if (task.enabled) return <Badge variant="secondary">{t('taskStateEnabled')}</Badge>
  return <Badge variant="outline">{t('taskStateDisabled')}</Badge>
}

function triggerSourceLabel(source: ScheduledTaskTriggerSource, t: (key: string) => string): string {
  return source === 'manual' ? t('taskTriggerManual') : t('taskTriggerScheduled')
}

export function SchedulerSection() {
  const { t, i18n } = useTranslation('settings')
  const [tasks, setTasks] = useState<ScheduledTask[]>([])
  const [tasksLoading, setTasksLoading] = useState(true)
  const [tasksError, setTasksError] = useState('')
  const [selectedCode, setSelectedCode] = useState('')
  const [runs, setRuns] = useState<ScheduledTaskRun[]>([])
  const [runsLoading, setRunsLoading] = useState(false)
  const [runsError, setRunsError] = useState('')
  const [statusFilter, setStatusFilter] = useState<RunStatusFilter>('all')
  const [triggerSourceFilter, setTriggerSourceFilter] = useState<TriggerSourceFilter>('all')
  const [actionMessage, setActionMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [pendingCode, setPendingCode] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [dialogTask, setDialogTask] = useState<ScheduledTask | null>(null)
  const [dialogForm, setDialogForm] = useState<EditFormState | null>(null)
  const [dialogError, setDialogError] = useState('')
  const [saving, setSaving] = useState(false)
  const [refreshing, setRefreshing] = useState(false)

  const selectedTask = tasks.find((task) => task.code === selectedCode) ?? null

  const loadTasks = useCallback(async ({ background = false }: { background?: boolean } = {}) => {
    if (!background) {
      setTasksLoading(true)
      setTasksError('')
    }
    try {
      const items = await scheduledTasksApi.list()
      setTasks(items)
      setTasksError('')
      setSelectedCode((current) => {
        if (current && items.some((task) => task.code === current)) return current
        return items[0]?.code ?? ''
      })
    } catch (err) {
      if (!background) {
        setTasksError(err instanceof Error ? err.message : t('taskLoadFailed'))
      }
    } finally {
      if (!background) {
        setTasksLoading(false)
      }
    }
  }, [t])

  const loadRuns = useCallback(async ({ background = false }: { background?: boolean } = {}) => {
    if (!selectedCode) {
      setRuns([])
      return
    }
    if (!background) {
      setRunsLoading(true)
      setRunsError('')
    }
    try {
      const page = await scheduledTasksApi.listRuns({
        task_code: selectedCode,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        trigger_source: triggerSourceFilter !== 'all' ? triggerSourceFilter : undefined,
        page: 1,
        page_size: 20,
      })
      setRuns(page.items)
      setRunsError('')
    } catch (err) {
      if (!background) {
        setRunsError(err instanceof Error ? err.message : t('taskRunsLoadFailed'))
      }
    } finally {
      if (!background) {
        setRunsLoading(false)
      }
    }
  }, [selectedCode, statusFilter, t, triggerSourceFilter])

  useEffect(() => {
    void loadTasks()
  }, [loadTasks])

  useEffect(() => {
    if (!selectedCode) return
    void loadRuns()
    const id = window.setInterval(() => {
      void loadTasks({ background: true })
      void loadRuns({ background: true })
    }, 5000)
    return () => window.clearInterval(id)
  }, [loadRuns, loadTasks, selectedCode])

  async function handleRefresh() {
    setRefreshing(true)
    try {
      await Promise.all([
        loadTasks({ background: true }),
        loadRuns({ background: true }),
      ])
    } finally {
      setRefreshing(false)
    }
  }

  function openEditDialog(task: ScheduledTask) {
    setDialogTask(task)
    setDialogForm({
      cronSpec: task.cron_spec,
      enabled: task.enabled,
      runTimeoutSeconds: String(task.run_timeout_seconds),
    })
    setDialogError('')
    setDialogOpen(true)
  }

  async function handleToggle(task: ScheduledTask) {
    setPendingCode(task.code)
    setActionMessage(null)
    try {
      await scheduledTasksApi.update(task.code, { enabled: !task.enabled })
      setActionMessage({
        type: 'success',
        text: !task.enabled ? t('taskEnableSuccess') : t('taskDisableSuccess'),
      })
      await loadTasks({ background: true })
      await loadRuns({ background: true })
    } catch (err) {
      setActionMessage({
        type: 'error',
        text: err instanceof Error ? err.message : t('taskUpdateFailed'),
      })
    } finally {
      setPendingCode('')
    }
  }

  async function handleTrigger(task: ScheduledTask) {
    setPendingCode(task.code)
    setActionMessage(null)
    setSelectedCode(task.code)
    try {
      await scheduledTasksApi.trigger(task.code)
      setActionMessage({ type: 'success', text: t('taskTriggerAccepted') })
      await loadTasks({ background: true })
      await loadRuns({ background: true })
    } catch (err) {
      setActionMessage({
        type: 'error',
        text: err instanceof Error ? err.message : t('taskTriggerFailed'),
      })
    } finally {
      setPendingCode('')
    }
  }

  async function handleSaveDialog() {
    if (!dialogTask || !dialogForm) return
    setSaving(true)
    setDialogError('')
    setActionMessage(null)
    try {
      await scheduledTasksApi.update(dialogTask.code, {
        cron_spec: dialogForm.cronSpec.trim(),
        enabled: dialogForm.enabled,
        run_timeout_seconds: Number(dialogForm.runTimeoutSeconds),
      })
      setDialogOpen(false)
      setActionMessage({ type: 'success', text: t('taskUpdateSuccess') })
      await loadTasks({ background: true })
      await loadRuns({ background: true })
    } catch (err) {
      setDialogError(err instanceof Error ? err.message : t('taskUpdateFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      {(tasksError || runsError || actionMessage) && (
        <div className="space-y-3">
          {tasksError && <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">{tasksError}</div>}
          {runsError && <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">{runsError}</div>}
          {actionMessage && (
            <div className={cn(
              'rounded-lg px-4 py-3 text-sm',
              actionMessage.type === 'success' ? 'bg-primary/10 text-primary' : 'bg-error-container text-error'
            )}>
              {actionMessage.text}
            </div>
          )}
        </div>
      )}

      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary/10 text-primary">
                <Clock3 className="h-5 w-5" />
              </div>
              <div>
                <CardTitle className="text-lg font-bold tracking-tight">{t('taskCenterTitle')}</CardTitle>
                <CardDescription>{t('taskCenterDescription')}</CardDescription>
              </div>
            </div>
            <Button variant="outline" size="sm" onClick={() => void handleRefresh()} disabled={tasksLoading || runsLoading || refreshing}>
              <RefreshCw className={cn('mr-2 h-4 w-4', refreshing && 'animate-spin')} />
              {t('taskRefresh')}
            </Button>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {tasksLoading ? (
            <div className="px-4 py-8 text-center text-sm text-on-surface-variant">{t('taskLoading')}</div>
          ) : tasks.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-on-surface-variant">{t('taskEmpty')}</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('taskColName')}</TableHead>
                  <TableHead>{t('taskColCode')}</TableHead>
                  <TableHead>{t('taskColCron')}</TableHead>
                  <TableHead>{t('taskColState')}</TableHead>
                  <TableHead>{t('taskColNextRun')}</TableHead>
                  <TableHead>{t('taskColLastResult')}</TableHead>
                  <TableHead className="text-right">{t('taskColActions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tasks.map((task) => {
                  const busy = pendingCode === task.code
                  const isSelected = task.code === selectedCode
                  return (
                    <TableRow
                      key={task.code}
                      className={cn('cursor-pointer', isSelected && 'bg-surface-container-low')}
                      onClick={() => setSelectedCode(task.code)}
                    >
                      <TableCell>
                        <div>
                          <p className="text-sm font-medium text-on-surface">{task.name}</p>
                          <p className="text-xs text-on-surface-variant">{task.description || t('taskNoDescription')}</p>
                        </div>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-on-surface-variant">{task.code}</TableCell>
                      <TableCell className="font-mono text-xs text-on-surface">{task.cron_spec}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2 flex-wrap">
                          {taskStateBadge(task, t)}
                        </div>
                      </TableCell>
                      <TableCell className="text-sm text-on-surface">
                        {formatDateTime(task.next_run_at, i18n.language, t('taskNever'))}
                      </TableCell>
                      <TableCell>{taskResultBadge(task.last_result, t)}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={(event) => {
                              event.stopPropagation()
                              openEditDialog(task)
                            }}
                            disabled={!task.registered}
                          >
                            <Pencil className="mr-2 h-4 w-4" />
                            {t('taskEdit')}
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={(event) => {
                              event.stopPropagation()
                              void handleToggle(task)
                            }}
                            disabled={busy || !task.registered}
                          >
                            {task.enabled ? <PowerOff className="mr-2 h-4 w-4" /> : <Power className="mr-2 h-4 w-4" />}
                            {task.enabled ? t('taskDisable') : t('taskEnable')}
                          </Button>
                          <Button
                            size="sm"
                            onClick={(event) => {
                              event.stopPropagation()
                              void handleTrigger(task)
                            }}
                            disabled={busy || !task.registered || task.state === 'running'}
                          >
                            {busy ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Play className="mr-2 h-4 w-4" />}
                            {t('taskRunNow')}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <div>
              <CardTitle className="text-lg font-bold tracking-tight">{t('taskRunsTitle')}</CardTitle>
              <CardDescription>
                {selectedTask ? t('taskRunsDescriptionSelected', { name: selectedTask.name }) : t('taskRunsDescription')}
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Select value={statusFilter} onValueChange={(value) => setStatusFilter(value as RunStatusFilter)}>
                <SelectTrigger className="h-9 w-36 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{t('taskFilterAllStatus')}</SelectItem>
                  <SelectItem value="running">{t('taskResultRunning')}</SelectItem>
                  <SelectItem value="success">{t('taskResultSuccess')}</SelectItem>
                  <SelectItem value="failed">{t('taskResultFailed')}</SelectItem>
                  <SelectItem value="skipped">{t('taskResultSkipped')}</SelectItem>
                </SelectContent>
              </Select>
              <Select value={triggerSourceFilter} onValueChange={(value) => setTriggerSourceFilter(value as TriggerSourceFilter)}>
                <SelectTrigger className="h-9 w-36 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{t('taskFilterAllSource')}</SelectItem>
                  <SelectItem value="manual">{t('taskTriggerManual')}</SelectItem>
                  <SelectItem value="scheduled">{t('taskTriggerScheduled')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {!selectedTask ? (
            <div className="px-4 py-8 text-center text-sm text-on-surface-variant">{t('taskSelectHint')}</div>
          ) : runsLoading ? (
            <div className="px-4 py-8 text-center text-sm text-on-surface-variant">{t('taskRunsLoading')}</div>
          ) : runs.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-on-surface-variant">{t('taskRunsEmpty')}</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('taskRunColStartedAt')}</TableHead>
                  <TableHead>{t('taskRunColDuration')}</TableHead>
                  <TableHead>{t('taskRunColStatus')}</TableHead>
                  <TableHead>{t('taskRunColSource')}</TableHead>
                  <TableHead>{t('taskRunColSummary')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {runs.map((run) => (
                  <TableRow key={run.id}>
                    <TableCell className="text-sm text-on-surface">
                      {formatDateTime(run.started_at, i18n.language, t('taskNever'))}
                    </TableCell>
                    <TableCell className="text-sm text-on-surface-variant">{formatDuration(run.duration_ms)}</TableCell>
                    <TableCell>{taskResultBadge(run.status, t)}</TableCell>
                    <TableCell className="text-sm text-on-surface-variant">
                      {triggerSourceLabel(run.trigger_source, t)}
                    </TableCell>
                    <TableCell>
                      <div>
                        <p className="text-sm text-on-surface">{run.summary || t('taskRunNoSummary')}</p>
                        {run.error_message && (
                          <p className="mt-1 text-xs text-error">{run.error_message}</p>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{t('taskEditDialogTitle')}</DialogTitle>
            <DialogDescription>{dialogTask?.name || t('taskEditDialogDescription')}</DialogDescription>
          </DialogHeader>
          {dialogForm && (
            <div className="space-y-4">
              <div className="space-y-1.5">
                <Label htmlFor="task-cron-spec">{t('taskEditCronLabel')}</Label>
                <Input
                  id="task-cron-spec"
                  value={dialogForm.cronSpec}
                  onChange={(event) => setDialogForm((current) => current ? { ...current, cronSpec: event.target.value } : current)}
                  placeholder="0 8 * * *"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="task-run-timeout">{t('taskEditTimeoutLabel')}</Label>
                <Input
                  id="task-run-timeout"
                  type="number"
                  min={1}
                  value={dialogForm.runTimeoutSeconds}
                  onChange={(event) => setDialogForm((current) => current ? { ...current, runTimeoutSeconds: event.target.value } : current)}
                />
              </div>
              <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
                <div>
                  <p className="text-sm font-medium text-on-surface">{t('taskEditEnabledLabel')}</p>
                  <p className="text-xs text-on-surface-variant">{t('taskEditEnabledHint')}</p>
                </div>
                <Checkbox
                  checked={dialogForm.enabled}
                  onCheckedChange={(checked) => setDialogForm((current) => current ? { ...current, enabled: checked === true } : current)}
                />
              </div>
              {dialogError && (
                <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">{dialogError}</div>
              )}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)} disabled={saving}>
              {t('taskEditCancel')}
            </Button>
            <Button onClick={() => void handleSaveDialog()} disabled={saving || !dialogForm}>
              {saving && <Loader2 className="h-4 w-4 animate-spin" />}
              {t('taskEditSave')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
