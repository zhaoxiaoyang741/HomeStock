import { useTranslation } from 'react-i18next'

export default function ShoppingPage() {
  const { t } = useTranslation('nav')
  return <div className="p-6">{t('shopping')}</div>
}
