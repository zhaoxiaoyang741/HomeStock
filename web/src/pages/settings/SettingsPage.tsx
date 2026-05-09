import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Radio, Brain, MessageSquare } from 'lucide-react'
import { cn } from '@/lib/utils'
import { FeishuBotSection } from './FeishuBotSection'
import { ModelConfigSection } from './ModelConfigSection'
import { Card, CardContent } from '@/components/ui/card'

type SectionId = 'channels' | 'models'

const SECTION_ITEMS: { id: SectionId; labelKey: string; icon: typeof Radio }[] = [
  { id: 'channels', labelKey: 'navChannels', icon: Radio },
  { id: 'models', labelKey: 'navModels', icon: Brain },
]

export default function SettingsPage() {
  const { t } = useTranslation('settings')
  const [activeSection, setActiveSection] = useState<SectionId>('channels')

  return (
    <div className="flex flex-col h-full gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">{t('settings:subtitle')}</p>
        </div>
      </div>

      {/* Two-panel layout */}
      <div className="flex flex-1 min-h-0 rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
        {/* Panel 1: Section nav */}
        <nav className="shrink-0 border-r border-outline-variant/20 flex flex-col py-2 w-40">
          {SECTION_ITEMS.map((item) => (
            <button
              key={item.id}
              onClick={() => setActiveSection(item.id)}
              className={cn(
                'flex items-center gap-3 w-full px-4 py-3 text-sm transition-colors',
                'text-on-surface-variant hover:bg-surface-container',
                activeSection === item.id &&
                  'bg-surface-container text-on-surface font-medium border-l-2 border-primary'
              )}
            >
              <item.icon className="w-4 h-4 shrink-0" />
              <span>{t(`settings:${item.labelKey}`)}</span>
            </button>
          ))}
        </nav>

        {/* Panel 2: Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {activeSection === 'channels' ? (
            <>
              {/* Feishu Bot */}
              <FeishuBotSection />

              {/* WeChat Bot placeholder */}
              <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
                <CardContent className="p-6">
                  <div className="flex items-center gap-3">
                    <MessageSquare className="w-5 h-5 text-on-surface-variant/40" />
                    <div>
                      <p className="text-sm font-medium text-on-surface">{t('navWechat')}</p>
                      <p className="text-xs text-on-surface-variant mt-0.5">{t('channelPlaceholder')}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </>
          ) : (
            <ModelConfigSection />
          )}
        </div>
      </div>
    </div>
  )
}
