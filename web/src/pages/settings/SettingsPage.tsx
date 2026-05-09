import { useTranslation } from 'react-i18next'
import { FeishuBotSection } from './FeishuBotSection'

export default function SettingsPage() {
  const { t } = useTranslation('settings')

  return (
    <div className="flex flex-col h-full gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('settings:title')}</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">{t('settings:subtitle')}</p>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 min-h-0 rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden p-6">
        <FeishuBotSection />
      </div>
    </div>
  )
}
