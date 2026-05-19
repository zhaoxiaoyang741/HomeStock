import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock, Loader2, Save } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { getCronConfig, updateCronConfig } from '@/api/cron'
import type { CronConfig } from '@/api/cron'

export function CronConfigSection() {
  const { t } = useTranslation('settings')
  const [config, setConfig] = useState<CronConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState('')

  // Editable fields
  const [enabled, setEnabled] = useState(false)
  const [intervalDays, setIntervalDays] = useState(7)
  const [pollInterval, setPollInterval] = useState('')
  const [notifyEnabled, setNotifyEnabled] = useState(false)
  const [notifyTimeStart, setNotifyTimeStart] = useState('')
  const [notifyTimeEnd, setNotifyTimeEnd] = useState('')

  const fetchConfig = async () => {
    try {
      const cfg = await getCronConfig()
      setConfig(cfg)
      setEnabled(cfg.enabled)
      setIntervalDays(cfg.expiry_check_interval_days)
      setPollInterval(cfg.expiry_check_poll_interval)
      setNotifyEnabled(cfg.notify_enabled)
      setNotifyTimeStart(cfg.notify_time_start)
      setNotifyTimeEnd(cfg.notify_time_end)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('cronSaveFailed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchConfig()
  }, [])

  async function handleSave() {
    setSaving(true)
    setError('')
    setSaveSuccess('')
    try {
      await updateCronConfig({
        enabled,
        expiry_check_interval_days: intervalDays,
        expiry_check_poll_interval: pollInterval || undefined,
        notify_enabled: notifyEnabled,
        notify_time_start: notifyTimeStart || undefined,
        notify_time_end: notifyTimeEnd || undefined,
      })
      setSaveSuccess(t('cronSaveSuccess'))
      await fetchConfig()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('cronSaveFailed'))
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="w-6 h-6 text-on-surface-variant animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-3">
            <Clock className="w-5 h-5 text-primary" />
            <div>
              <CardTitle className="text-lg font-bold tracking-tight">{t('cronSectionTitle')}</CardTitle>
              <CardDescription>{t('cronSectionDescription')}</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-5">
          <Separator />

          {/* Error message */}
          {error && (
            <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">
              {error}
            </div>
          )}

          {/* Success message */}
          {saveSuccess && (
            <div className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">
              {saveSuccess}
            </div>
          )}

          {/* ── Cron Schedule Settings ── */}
          <h4 className="text-sm font-medium text-on-surface">{t('cronSectionTitle')}</h4>

          {/* Cron enabled toggle */}
          <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
            <div>
              <p className="text-sm font-medium text-on-surface">{t('cronEnabled')}</p>
              <p className="text-xs text-on-surface-variant">{t('cronEnabledHint')}</p>
            </div>
            <Checkbox
              checked={enabled}
              onCheckedChange={(checked) => setEnabled(checked === true)}
            />
          </div>

          {/* Expiry check interval days */}
          <div className="space-y-1.5">
            <Label htmlFor="cron-interval-days">{t('cronIntervalDays')}</Label>
            <Input
              id="cron-interval-days"
              type="number"
              min={1}
              value={intervalDays}
              onChange={(e) => setIntervalDays(parseInt(e.target.value) || 7)}
            />
            <p className="text-xs text-on-surface-variant">{t('cronIntervalDaysHint')}</p>
          </div>

          {/* Poll interval */}
          <div className="space-y-1.5">
            <Label htmlFor="cron-poll-interval">{t('cronPollInterval')}</Label>
            <Input
              id="cron-poll-interval"
              placeholder={config?.expiry_check_poll_interval || '30m'}
              value={pollInterval}
              onChange={(e) => setPollInterval(e.target.value)}
            />
            <p className="text-xs text-on-surface-variant">{t('cronPollIntervalHint')}</p>
          </div>

          <Separator />

          {/* ── Notification Settings ── */}
          <h4 className="text-sm font-medium text-on-surface">{t('notifySectionTitle')}</h4>

          {/* Notify enabled toggle */}
          <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
            <div>
              <p className="text-sm font-medium text-on-surface">{t('notifyEnabled')}</p>
              <p className="text-xs text-on-surface-variant">{t('notifyEnabledHint')}</p>
            </div>
            <Checkbox
              checked={notifyEnabled}
              onCheckedChange={(checked) => setNotifyEnabled(checked === true)}
            />
          </div>

          {/* Notify time start */}
          <div className="space-y-1.5">
            <Label htmlFor="notify-time-start">{t('notifyTimeStart')}</Label>
            <Input
              id="notify-time-start"
              type="time"
              value={notifyTimeStart}
              onChange={(e) => setNotifyTimeStart(e.target.value)}
            />
          </div>

          {/* Notify time end */}
          <div className="space-y-1.5">
            <Label htmlFor="notify-time-end">{t('notifyTimeEnd')}</Label>
            <Input
              id="notify-time-end"
              type="time"
              value={notifyTimeEnd}
              onChange={(e) => setNotifyTimeEnd(e.target.value)}
            />
          </div>

          {/* Save button */}
          <Button
            size="sm"
            onClick={() => void handleSave()}
            disabled={saving}
          >
            {saving ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-2 h-4 w-4" />
            )}
            {saving ? t('modelSaving') : t('common:save')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
