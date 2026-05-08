import { useEffect, useMemo, useState } from 'react'
import { RefreshCw, Save, Clock, Bell, Webhook } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { useSystemSettings } from '@/hooks/useSystemSettings'
import type { SystemSettings, UpdateSystemSettingsPayload } from '@/types/systemSettings'
import { SchedulerSection } from './SchedulerSection'
import { ReminderSection } from './ReminderSection'
import { ChannelSection } from './ChannelSection'

type SectionId = 'scheduler' | 'reminder' | 'channel'

type NavItem = { id: SectionId; labelKey: string; icon: LucideIcon }

const NAV_ITEMS: NavItem[] = [
  { id: 'scheduler', labelKey: 'navScheduler', icon: Clock },
  { id: 'reminder',  labelKey: 'navReminder',  icon: Bell },
  { id: 'channel',   labelKey: 'navChannel',   icon: Webhook },
]

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
  const [activeSection, setActiveSection] = useState<SectionId>('scheduler')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const { settings, loading, error: settingsError, refreshSettings, updateSettings } = useSystemSettings()
  const [initialSettings, setInitialSettings] = useState<SystemSettings | null>(null)
  const [form, setForm] = useState<FormState | null>(null)

  useEffect(() => {
    if (!settings) return
    if (initialSettings?.version === settings.version) return
    setInitialSettings(settings)
    setForm(formFromSettings(settings))
    setError('')
  }, [settings, initialSettings?.version])

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
      const saved = await updateSettings(payload)
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

  function patchForm(patch: Partial<FormState>) {
    setForm((current) => current ? { ...current, ...patch } : current)
  }

  const isFormSection = activeSection === 'reminder' || activeSection === 'channel'

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
      {/* Page header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">{t('settings:subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void refreshSettings()} disabled={loading || saving}>
            <RefreshCw className={cn('mr-2 h-4 w-4', loading && form ? 'animate-spin' : '')} />
            {t('common:refresh')}
          </Button>
          {isFormSection && (
            <>
              <Button variant="outline" size="sm" onClick={handleReset} disabled={!dirty || saving}>
                {t('settings:reset')}
              </Button>
              <Button size="sm" onClick={() => void handleSave()} disabled={!dirty || saving || loading || !form}>
                <Save className="mr-2 h-4 w-4" />
                {saving ? t('settings:saving') : t('common:save')}
              </Button>
            </>
          )}
        </div>
      </div>

      {(error || settingsError) && (
        <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">
          {error || settingsError || t('settings:loadFailed')}
        </div>
      )}

      {/* Main layout: sidebar + content */}
      <div className="flex flex-1 min-h-0 rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
        {/* Sidebar nav */}
        <nav className="shrink-0 border-r border-outline-variant/20 flex flex-col py-2 w-12 md:w-50">
          {NAV_ITEMS.map((item) => (
            <button
              key={item.id}
              onClick={() => setActiveSection(item.id)}
              className={cn(
                'flex items-center gap-3 w-full px-3 md:px-4 py-3 text-sm transition-colors',
                'text-on-surface-variant hover:bg-surface-container',
                activeSection === item.id &&
                  'bg-surface-container text-on-surface font-medium border-l-2 border-primary'
              )}
            >
              <item.icon className="w-4 h-4 shrink-0" />
              <span className="hidden md:block">{t(`settings:${item.labelKey}`)}</span>
            </button>
          ))}
        </nav>

        {/* Content panel */}
        <div className="flex-1 overflow-y-auto p-6">
          {activeSection === 'scheduler' && <SchedulerSection />}

          {activeSection === 'reminder' && form && (
            <ReminderSection
              form={{ remindDays: form.remindDays, notifyEnabled: form.notifyEnabled }}
              onChange={patchForm}
            />
          )}

          {activeSection === 'channel' && form && initialSettings && (
            <ChannelSection
              form={{ feishuWebhookInput: form.feishuWebhookInput, webhookMode: form.webhookMode }}
              initialSettings={initialSettings}
              onChange={patchForm}
            />
          )}

          {/* Metadata footer (form sections only) */}
          {isFormSection && initialSettings && (
            <div className="mt-6 rounded-xl border border-outline-variant/20 bg-surface-container px-4 py-4">
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
      </div>
    </div>
  )
}
