import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock, Send, Key, Lock, Tags } from 'lucide-react'
import { cn } from '@/lib/utils'
import { OutboundSection } from './OutboundSection'
import { ApiKeySection } from './ApiKeySection'
import { CronConfigSection } from './CronConfigSection'
import { ChangePasswordSection } from './ChangePasswordSection'
import { CategorySection } from './CategorySection'

type SectionId = 'outbound' | 'apikey' | 'cron' | 'password' | 'category'

const SECTION_ITEMS: { id: SectionId; labelKey: string; icon: typeof Send }[] = [
  { id: 'outbound', labelKey: 'navOutbound', icon: Send },
  { id: 'apikey', labelKey: 'navApiKey', icon: Key },
  { id: 'cron', labelKey: 'navCron', icon: Clock },
  { id: 'password', labelKey: 'navPassword', icon: Lock },
  { id: 'category', labelKey: 'navCategory', icon: Tags },
]

export default function SettingsPage() {
  const { t } = useTranslation('settings')
  const [activeSection, setActiveSection] = useState<SectionId>('outbound')

  return (
    <div className="flex h-full min-h-0 flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
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

        <div className="min-h-0 flex-1 overflow-y-auto p-6">
          {activeSection === 'outbound' ? (
            <OutboundSection />
          ) : activeSection === 'apikey' ? (
            <ApiKeySection />
          ) : activeSection === 'cron' ? (
            <CronConfigSection />
          ) : activeSection === 'password' ? (
            <ChangePasswordSection />
          ) : (
            <CategorySection />
          )}
        </div>
      </div>
    </div>
  )
}
