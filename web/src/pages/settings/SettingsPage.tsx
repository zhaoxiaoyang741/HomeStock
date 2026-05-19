import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock, Radio, Brain } from 'lucide-react'
import { cn } from '@/lib/utils'
import { getModelList } from '@/api/models'
import { FeishuBotSection } from './FeishuBotSection'
import { WechatBotSection } from './WechatBotSection'
import { ModelConfigSection } from './ModelConfigSection'
import { CronConfigSection } from './CronConfigSection'

type SectionId = 'channels' | 'models' | 'cron'

const SECTION_ITEMS: { id: SectionId; labelKey: string; icon: typeof Radio }[] = [
  { id: 'channels', labelKey: 'navChannels', icon: Radio },
  { id: 'cron', labelKey: 'navCron', icon: Clock },
  { id: 'models', labelKey: 'navModels', icon: Brain },
]

export default function SettingsPage() {
  const { t } = useTranslation('settings')
  const [activeSection, setActiveSection] = useState<SectionId>('channels')
  const [activeChannel, setActiveChannel] = useState<'feishu' | 'wechat' | null>(null)
  const [lastReloadTime, setLastReloadTime] = useState('')
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchReloadTime = async () => {
    try {
      const data = await getModelList()
      setLastReloadTime(data.last_reload_time)
    } catch {
      // Child sections show their own error state.
    }
  }

  useEffect(() => {
    void fetchReloadTime()
    pollingRef.current = setInterval(() => {
      void fetchReloadTime()
    }, 10000)

    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current)
    }
  }, [])

  return (
    <div className="flex h-full min-h-0 flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
          <p className="mt-0.5 text-sm text-on-surface-variant">{t('settings:subtitle')}</p>
        </div>
      </div>

      <div className="flex min-h-0 flex-1 overflow-hidden rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <nav className="flex w-40 shrink-0 flex-col border-r border-outline-variant/20 py-2">
          {SECTION_ITEMS.map((item) => (
            <button
              key={item.id}
              onClick={() => setActiveSection(item.id)}
              className={cn(
                'flex w-full items-center gap-3 px-4 py-3 text-sm transition-colors',
                'text-on-surface-variant hover:bg-surface-container',
                activeSection === item.id &&
                  'border-l-2 border-primary bg-surface-container font-medium text-on-surface',
              )}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              <span>{t(`settings:${item.labelKey}`)}</span>
            </button>
          ))}
        </nav>

        <div className="flex min-h-0 flex-1 flex-col p-6">
          {lastReloadTime && (
            <div className="flex shrink-0 items-center gap-2 border-b border-outline-variant/20 pb-2 text-xs text-on-surface-variant">
              <Clock className="h-3.5 w-3.5 shrink-0" />
              <span>{t('lastReloadTime')}: {lastReloadTime}</span>
            </div>
          )}

          <div className="min-h-0 flex-1 overflow-y-auto pt-6">
            {activeSection === 'channels' ? (
              <div className="space-y-6">
                <FeishuBotSection
                  isActive={activeChannel === 'feishu'}
                  onActivate={() => setActiveChannel('feishu')}
                  onDeactivate={() => setActiveChannel(null)}
                />
                <WechatBotSection
                  isActive={activeChannel === 'wechat'}
                  onActivate={() => setActiveChannel('wechat')}
                  onDeactivate={() => setActiveChannel(null)}
                />
              </div>
            ) : activeSection === 'cron' ? (
              <CronConfigSection />
            ) : (
              <ModelConfigSection />
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
