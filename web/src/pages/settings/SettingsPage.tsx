import { useEffect, useMemo, useState } from 'react'
import { RefreshCw, Save, Settings2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { systemSettingsApi } from '@/api/systemSettings'
import type { SystemSettings, UpdateSystemSettingsPayload } from '@/types/systemSettings'

type FormState = {
  remindDays: string
  checkTime: string
  notifyEnabled: boolean
  feishuWebhookInput: string
  webhookMode: 'keep' | 'replace' | 'clear'
}

function formFromSettings(settings: SystemSettings): FormState {
  return {
    remindDays: String(settings.reminder.remind_days),
    checkTime: settings.reminder.check_time,
    notifyEnabled: settings.notify.enabled,
    feishuWebhookInput: '',
    webhookMode: 'keep',
  }
}

export default function SettingsPage() {
  const { t, i18n } = useTranslation(['settings', 'common', 'history'])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [initialSettings, setInitialSettings] = useState<SystemSettings | null>(null)
  const [form, setForm] = useState<FormState | null>(null)

  async function loadSettings() {
    setLoading(true)
    setError('')
    try {
      const next = await systemSettingsApi.get()
      setInitialSettings(next)
      setForm(formFromSettings(next))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('settings:loadFailed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadSettings()
  }, [])

  const dirty = useMemo(() => {
    if (!initialSettings || !form) return false
    return (
      form.remindDays !== String(initialSettings.reminder.remind_days) ||
      form.checkTime !== initialSettings.reminder.check_time ||
      form.notifyEnabled !== initialSettings.notify.enabled ||
      form.webhookMode !== 'keep' ||
      form.feishuWebhookInput.trim() !== ''
    )
  }, [form, initialSettings])

  async function handleSave() {
    if (!initialSettings || !form) return
    setSaving(true)
    setError('')
    try {
      const payload: UpdateSystemSettingsPayload = {
        version: initialSettings.version,
        reminder: {
          remind_days: Number(form.remindDays),
          check_time: form.checkTime,
        },
        notify: {
          enabled: form.notifyEnabled,
          feishu_webhook_mode: form.webhookMode,
          feishu_webhook: form.webhookMode === 'replace' ? form.feishuWebhookInput.trim() : undefined,
        },
      }
      const saved = await systemSettingsApi.update(payload)
      setInitialSettings(saved)
      setForm(formFromSettings(saved))
    } catch (err) {
      if (err instanceof Error && err.message === 'system settings version conflict') {
        setError(t('settings:errorConflict'))
      } else {
        setError(err instanceof Error ? err.message : t('settings:saveFailed'))
      }
    } finally {
      setSaving(false)
    }
  }

  function handleReset() {
    if (!initialSettings) return
    setForm(formFromSettings(initialSettings))
    setError('')
  }

  if (loading && !form) {
    return (
      <div className="flex flex-col h-full gap-6">
        <div className="flex items-center justify-between gap-4 flex-wrap">
          <div>
            <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
            <p className="text-sm text-on-surface-variant mt-0.5">{t('settings:subtitle')}</p>
          </div>
        </div>
        <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest px-4 py-8 text-sm text-on-surface-variant shadow-sm">
          {t('common:loading')}
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full gap-6">
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">{t('settings:subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void loadSettings()} disabled={loading || saving}>
            <RefreshCw className={`mr-2 h-4 w-4 ${(loading && form) ? 'animate-spin' : ''}`} />
            {t('common:refresh')}
          </Button>
          <Button variant="outline" size="sm" onClick={handleReset} disabled={!dirty || saving}>
            {t('settings:reset')}
          </Button>
          <Button size="sm" onClick={() => void handleSave()} disabled={!dirty || saving || loading || !form}>
            <Save className="mr-2 h-4 w-4" />
            {saving ? t('settings:saving') : t('common:save')}
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">
          {error}
        </div>
      )}

      {form && initialSettings && (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
          <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
            <CardHeader className="pb-4">
              <div className="flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary/10 text-primary">
                  <Settings2 className="h-5 w-5" />
                </div>
                <div>
                  <CardTitle className="text-lg font-bold tracking-tight">{t('settings:reminderTitle')}</CardTitle>
                  <CardDescription>{t('settings:reminderDescription')}</CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="settings-remind-days">{t('settings:fieldRemindDays')}</Label>
                  <Input
                    id="settings-remind-days"
                    type="number"
                    min={0}
                    max={30}
                    value={form.remindDays}
                    onChange={(event) => setForm((current) => current ? { ...current, remindDays: event.target.value } : current)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="settings-check-time">{t('settings:fieldCheckTime')}</Label>
                  <Input
                    id="settings-check-time"
                    type="time"
                    value={form.checkTime}
                    onChange={(event) => setForm((current) => current ? { ...current, checkTime: event.target.value } : current)}
                  />
                </div>
              </div>
              <div className="rounded-lg bg-surface-container px-4 py-3 text-sm text-on-surface-variant">
                {t('settings:reminderHint')}
              </div>
            </CardContent>
          </Card>

          <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
            <CardHeader className="pb-4">
              <CardTitle className="text-lg font-bold tracking-tight">{t('settings:notifyTitle')}</CardTitle>
              <CardDescription>{t('settings:notifyDescription')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
                <div>
                  <p className="text-sm font-medium text-on-surface">{t('settings:fieldNotifyEnabled')}</p>
                  <p className="text-xs text-on-surface-variant">{t('settings:notifyEnabledHint')}</p>
                </div>
                <Checkbox
                  checked={form.notifyEnabled}
                  onCheckedChange={(checked) => setForm((current) => current ? { ...current, notifyEnabled: checked === true } : current)}
                />
              </div>

              <Separator />

              <div className="space-y-3">
                <div className="flex items-center justify-between gap-3 flex-wrap">
                  <div className="space-y-1">
                    <Label htmlFor="settings-webhook">{t('settings:fieldWebhook')}</Label>
                    <p className="text-xs text-on-surface-variant">{t('settings:webhookHint')}</p>
                  </div>
                  <Badge variant={initialSettings.notify.feishu_webhook_configured ? 'outline' : 'secondary'}>
                    {initialSettings.notify.feishu_webhook_configured ? t('settings:webhookConfigured') : t('settings:webhookNotConfigured')}
                  </Badge>
                </div>

                {initialSettings.notify.feishu_webhook_configured && (
                  <div className="rounded-lg bg-surface-container px-3 py-2 text-sm text-on-surface-variant">
                    {t('settings:webhookMaskedLabel', { value: initialSettings.notify.feishu_webhook_masked })}
                  </div>
                )}

                <Input
                  id="settings-webhook"
                  type="url"
                  value={form.feishuWebhookInput}
                  placeholder={t('settings:webhookPlaceholder')}
                  onChange={(event) => {
                    const value = event.target.value
                    setForm((current) => {
                      if (!current) return current
                      return {
                        ...current,
                        feishuWebhookInput: value,
                        webhookMode: value.trim() ? 'replace' : 'keep',
                      }
                    })
                  }}
                />

                <div className="flex items-center gap-2 flex-wrap">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setForm((current) => current ? { ...current, feishuWebhookInput: '', webhookMode: 'clear' } : current)}
                    disabled={!initialSettings.notify.feishu_webhook_configured && form.feishuWebhookInput.trim() === ''}
                  >
                    {t('settings:clearWebhook')}
                  </Button>
                  {form.webhookMode === 'clear' && (
                    <span className="text-xs text-error">{t('settings:webhookClearPending')}</span>
                  )}
                  {form.webhookMode === 'replace' && (
                    <span className="text-xs text-primary">{t('settings:webhookReplacePending')}</span>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {initialSettings && (
        <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest px-4 py-4 shadow-sm">
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <div>
              <p className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('settings:lastUpdated')}</p>
              <p className="mt-1 text-sm text-on-surface">
                {initialSettings.updated_at
                  ? t('settings:lastUpdatedValue', {
                    value: new Date(initialSettings.updated_at).toLocaleString(i18n.language),
                    user: initialSettings.updated_by.user_name || initialSettings.updated_by.user_id || t('history:channelSystem'),
                  })
                  : t('settings:notSavedYet')}
              </p>
            </div>
            <Badge variant="secondary">{t('settings:versionBadge', { version: initialSettings.version })}</Badge>
          </div>
        </div>
      )}
    </div>
  )
}
