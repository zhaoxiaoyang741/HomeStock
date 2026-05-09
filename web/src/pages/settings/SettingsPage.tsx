import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Bot } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import { FeishuBotSection } from './FeishuBotSection'

type ChannelId = 'feishu'

type NavItem = {
  id: ChannelId
  labelKey: string
  icon: LucideIcon
}

const CHANNEL_ITEMS: NavItem[] = [
  { id: 'feishu', labelKey: 'navFeishu', icon: Bot },
]

export default function SettingsPage() {
  const { t } = useTranslation('settings')
  const [activeChannel, setActiveChannel] = useState<ChannelId>('feishu')

  return (
    <div className="flex flex-col h-full gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">{t('settings:subtitle')}</p>
        </div>
      </div>

      {/* Main layout: sidebar + content */}
      <div className="flex flex-1 min-h-0 rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
        {/* Left sidebar nav */}
        <nav className="shrink-0 border-r border-outline-variant/20 flex flex-col py-2 w-50">
          <div className="px-4 pb-2 pt-1">
            <p className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">
              {t('navChannel')}
            </p>
          </div>
          {CHANNEL_ITEMS.map((item) => (
            <button
              key={item.id}
              onClick={() => setActiveChannel(item.id)}
              className={cn(
                'flex items-center gap-3 w-full px-4 py-3 text-sm transition-colors',
                'text-on-surface-variant hover:bg-surface-container',
                activeChannel === item.id &&
                  'bg-surface-container text-on-surface font-medium border-l-2 border-primary'
              )}
            >
              <item.icon className="w-4 h-4 shrink-0" />
              <span>{t(`settings:${item.labelKey}`)}</span>
            </button>
          ))}
        </nav>

        {/* Content panel */}
        <div className="flex-1 overflow-y-auto p-6">
          {activeChannel === 'feishu' && <FeishuBotSection />}
        </div>
      </div>
    </div>
  )
}
