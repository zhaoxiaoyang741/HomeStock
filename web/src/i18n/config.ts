import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import zhCNCommon from './locales/zh-CN/common.json'
import zhCNNav from './locales/zh-CN/nav.json'
import zhCNHeader from './locales/zh-CN/header.json'
import zhCNInventory from './locales/zh-CN/inventory.json'
import zhCNHistory from './locales/zh-CN/history.json'
import zhCNSettings from './locales/zh-CN/settings.json'
import zhCNDashboard from './locales/zh-CN/dashboard.json'
import zhCNAuth from './locales/zh-CN/auth.json'

import enCommon from './locales/en/common.json'
import enNav from './locales/en/nav.json'
import enHeader from './locales/en/header.json'
import enInventory from './locales/en/inventory.json'
import enHistory from './locales/en/history.json'
import enSettings from './locales/en/settings.json'
import enDashboard from './locales/en/dashboard.json'
import enAuth from './locales/en/auth.json'

const savedLanguage = (() => {
  try {
    return localStorage.getItem('language') ?? 'zh-CN'
  } catch {
    return 'zh-CN'
  }
})()

i18n.use(initReactI18next).init({
  lng: savedLanguage,
  fallbackLng: 'zh-CN',
  defaultNS: 'common',
  resources: {
    'zh-CN': {
      common: zhCNCommon,
      nav: zhCNNav,
      header: zhCNHeader,
      inventory: zhCNInventory,
      history: zhCNHistory,
      settings: zhCNSettings,
      dashboard: zhCNDashboard,
      auth: zhCNAuth,
    },
    en: {
      common: enCommon,
      nav: enNav,
      header: enHeader,
      inventory: enInventory,
      history: enHistory,
      settings: enSettings,
      dashboard: enDashboard,
      auth: enAuth,
    },
  },
  interpolation: {
    escapeValue: false,
  },
})

export default i18n
