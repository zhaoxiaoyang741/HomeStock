import { useTranslation } from 'react-i18next'

export default function SettingsPage() {
  const { t } = useTranslation('nav')
  return <div className="p-6">{t('settings')}</div>
}
