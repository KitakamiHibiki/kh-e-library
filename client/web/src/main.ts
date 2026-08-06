import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import router from './router'
import { installI18n, getSavedLocale, elementLocales } from './locales'
import i18n from './locales'
import type { SupportedLocale } from './locales'

const app = createApp(App)

installI18n(app)
app.use(ElementPlus, { locale: elementLocales[getSavedLocale()] })
app.use(router)
app.mount('#app')

// Sync Element Plus locale when language changes
import { watch } from 'vue'
watch(
  () => i18n.global.locale.value,
  (val) => {
    const ep = app.config.globalProperties.$ELEMENT
    if (ep) ep.locale = elementLocales[val as SupportedLocale]
  }
)
