import { createI18n } from 'vue-i18n'
import viVN from './vi-VN'

export const defaultLocale = 'vi-VN'

export const i18n = createI18n({
  legacy: false,
  locale: defaultLocale,
  fallbackLocale: defaultLocale,
  messages: {
    [defaultLocale]: viVN
  }
})

export const t = i18n.global.t
