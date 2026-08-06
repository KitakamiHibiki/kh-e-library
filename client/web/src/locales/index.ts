import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import en from './en'
import elementZhCN from 'element-plus/es/locale/lang/zh-cn'
import elementEn from 'element-plus/es/locale/lang/en'
import type { App } from 'vue'

export type SupportedLocale = 'zh-CN' | 'en'

const LOCALE_KEY = 'kh-e-library-locale'

export function getSavedLocale(): SupportedLocale {
  const stored = localStorage.getItem(LOCALE_KEY)
  if (stored === 'zh-CN' || stored === 'en') return stored
  return 'zh-CN'
}

export function saveLocale(locale: SupportedLocale) {
  localStorage.setItem(LOCALE_KEY, locale)
}

export const elementLocales: Record<SupportedLocale, typeof elementZhCN> = {
  'zh-CN': elementZhCN,
  'en': elementEn,
}

const i18n = createI18n({
  legacy: false,
  locale: getSavedLocale(),
  fallbackLocale: 'zh-CN',
  messages: { 'zh-CN': zhCN, en },
})

export function installI18n(app: App) {
  app.use(i18n)
}

export default i18n
