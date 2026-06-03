import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock, Radio, Brain, Settings, Save, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { getModelList } from '@/api/models'
import { FeishuBotSection } from './FeishuBotSection'
import type { FeishuBotSectionHandle } from './FeishuBotSection'
import { WechatBotSection } from './WechatBotSection'
import type { WechatBotSectionHandle } from './WechatBotSection'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { ModelConfigSection } from './ModelConfigSection'
import { CronConfigSection } from './CronConfigSection'
import { ChangePasswordSection } from './ChangePasswordSection'
import { CategorySection } from './CategorySection'
import { InventoryConfigSection } from './InventoryConfigSection'

type SectionId = 'models' | 'channels' | 'cron' | 'system'

interface NavItem {
  id: SectionId
  labelKey: string
  icon: typeof Radio
}

export default function SettingsPage() {
  const { t } = useTranslation('settings')
  const [activeSection, setActiveSection] = useState<SectionId>('channels')
  const [activeChannel, setActiveChannel] = useState<'feishu' | 'wechat' | null>(null)
  const [lastReloadTime, setLastReloadTime] = useState('')
  const [savingChannels, setSavingChannels] = useState(false)
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const feishuRef = useRef<FeishuBotSectionHandle>(null)
  const wechatRef = useRef<WechatBotSectionHandle>(null)

  const navItems: NavItem[] = [
    { id: 'models', labelKey: 'navModels', icon: Brain },
    { id: 'channels', labelKey: 'navChannels', icon: Radio },
    { id: 'cron', labelKey: 'navCron', icon: Clock },
    { id: 'system', labelKey: 'navSystem', icon: Settings },
  ]

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

  const handleSaveChannels = async () => {
    setSavingChannels(true)
    try {
      await feishuRef.current?.save()
      await wechatRef.current?.save()
    } finally {
      setSavingChannels(false)
    }
  }

  function handleNavClick(id: SectionId) {
    setActiveSection(id)
  }

  function renderSection() {
    switch (activeSection) {
      case 'channels':
        return (
          <div className="space-y-6">
            <FeishuBotSection
              ref={feishuRef}
              isActive={activeChannel === 'feishu'}
              onActivate={() => setActiveChannel('feishu')}
              onDeactivate={() => setActiveChannel(null)}
            />
            <WechatBotSection
              ref={wechatRef}
              isActive={activeChannel === 'wechat'}
              onActivate={() => setActiveChannel('wechat')}
              onDeactivate={() => setActiveChannel(null)}
            />
            <div className="h-4" />
          </div>
        )
      case 'cron':
        return <CronConfigSection />
      case 'models':
        return <ModelConfigSection />
      case 'system':
        return (
          <div className="space-y-8">
            <div>
              <h2 className="text-base font-semibold text-on-surface mb-4 flex items-center gap-2">
                <span className="inline-block w-1 h-4 bg-primary rounded-full" />
                {t('navCategory')}
              </h2>
              <CategorySection />
            </div>
            <Separator />
            <div>
              <h2 className="text-base font-semibold text-on-surface mb-4 flex items-center gap-2">
                <span className="inline-block w-1 h-4 bg-primary rounded-full" />
                {t('navInventory')}
              </h2>
              <InventoryConfigSection />
            </div>
            <Separator />
            <div>
              <h2 className="text-base font-semibold text-on-surface mb-4 flex items-center gap-2">
                <span className="inline-block w-1 h-4 bg-primary rounded-full" />
                {t('navUser')}
              </h2>
              <ChangePasswordSection />
            </div>
          </div>
        )
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
        </div>
      </div>

      <div className="flex min-h-0 flex-1 overflow-hidden rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <nav className="flex w-44 shrink-0 flex-col border-r border-outline-variant/20 py-2">
          {navItems.map((item) => (
            <button
              key={item.id}
              onClick={() => handleNavClick(item.id)}
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
            {renderSection()}
          </div>

          {/* Floating save bar for channels section */}
          {activeSection === 'channels' && (
            <div className="flex shrink-0 items-center justify-end border-t border-outline-variant/20 bg-surface-container-lowest pt-4">
              <Button
                onClick={() => void handleSaveChannels()}
                disabled={savingChannels}
              >
                {savingChannels ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Save className="mr-2 h-4 w-4" />
                )}
                {t('common:save')}
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
