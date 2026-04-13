import { useTranslation } from 'react-i18next'

export default function HomedPage() {
  const { t } = useTranslation('nav')
  return <div className="p-6">{t('home')}</div>
}
