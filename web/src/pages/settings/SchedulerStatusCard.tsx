import { useCallback, useEffect, useState } from 'react'
import { Play, Clock, CheckCircle2, XCircle, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { schedulerApi } from '@/api/scheduler'
import type { SchedulerStatus } from '@/types/scheduler'
import { cn } from '@/lib/utils'

function formatDateTime(iso: string | null, locale: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString(locale)
}

export function SchedulerStatusCard() {
  const { t, i18n } = useTranslation('settings')
  const [status, setStatus] = useState<SchedulerStatus | null>(null)
  const [triggering, setTriggering] = useState(false)
  const [triggerMsg, setTriggerMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const fetchStatus = useCallback(async () => {
    try {
      const s = await schedulerApi.getStatus()
      setStatus(s)
    } catch {
      // silently ignore polling errors
    }
  }, [])

  useEffect(() => {
    void fetchStatus()
    const id = setInterval(() => void fetchStatus(), 10_000)
    return () => clearInterval(id)
  }, [fetchStatus])

  async function handleTrigger() {
    setTriggering(true)
    setTriggerMsg(null)
    try {
      await schedulerApi.trigger()
      setTriggerMsg({ type: 'success', text: t('schedulerTriggerSuccess') })
      await fetchStatus()
    } catch {
      setTriggerMsg({ type: 'error', text: t('schedulerTriggerFailed') })
    } finally {
      setTriggering(false)
    }
  }

  const stateLabel = status?.state === 'running' ? t('schedulerStateRunning') : t('schedulerStateIdle')
  const isRunning = status?.state === 'running'

  function resultBadge() {
    if (!status?.last_result) return <span className="text-sm text-on-surface-variant">{t('schedulerResultNone')}</span>
    if (status.last_result === 'success')
      return <Badge variant="default" className="gap-1"><CheckCircle2 className="w-3 h-3" />{t('schedulerResultSuccess')}</Badge>
    return <Badge variant="destructive" className="gap-1"><XCircle className="w-3 h-3" />{t('schedulerResultFailed')}</Badge>
  }

  return (
    <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div className="flex items-center gap-3">
            <div className={cn(
              'flex h-11 w-11 items-center justify-center rounded-full',
              isRunning ? 'bg-primary/10 text-primary' : 'bg-surface-container text-on-surface-variant'
            )}>
              {isRunning
                ? <Loader2 className="h-5 w-5 animate-spin" />
                : <Clock className="h-5 w-5" />}
            </div>
            <div>
              <CardTitle className="text-lg font-bold tracking-tight">{t('schedulerTitle')}</CardTitle>
              <CardDescription>{t('schedulerDescription')}</CardDescription>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant={isRunning ? 'default' : 'secondary'}>{stateLabel}</Badge>
            <Button
              size="sm"
              variant="outline"
              onClick={() => void handleTrigger()}
              disabled={triggering || isRunning}
            >
              {triggering
                ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />{t('schedulerTriggering')}</>
                : <><Play className="mr-2 h-4 w-4" />{t('schedulerTrigger')}</>}
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-3 gap-4 text-sm">
          <div>
            <p className="text-xs font-medium text-on-surface-variant mb-1">{t('schedulerLastRun')}</p>
            <p className="text-on-surface">{formatDateTime(status?.last_run_at ?? null, i18n.language)}</p>
          </div>
          <div>
            <p className="text-xs font-medium text-on-surface-variant mb-1">{t('schedulerNextRun')}</p>
            <p className="text-on-surface">{formatDateTime(status?.next_run_at ?? null, i18n.language)}</p>
          </div>
          <div>
            <p className="text-xs font-medium text-on-surface-variant mb-1">{t('schedulerLastResult')}</p>
            {resultBadge()}
          </div>
        </div>
        {status?.last_error && (
          <div className="mt-3 rounded-lg bg-error-container px-3 py-2 text-xs text-error">
            {status.last_error}
          </div>
        )}
        {triggerMsg && (
          <div className={cn(
            'mt-3 rounded-lg px-3 py-2 text-xs',
            triggerMsg.type === 'success' ? 'bg-primary/10 text-primary' : 'bg-error-container text-error'
          )}>
            {triggerMsg.text}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
