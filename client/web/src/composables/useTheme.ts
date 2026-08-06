import { ref, watch, onUnmounted } from 'vue'
import { getSettings } from '@/api'

export function useTheme() {
  const isDark = ref(false)
  let mediaQueryHandler: ((e: MediaQueryListEvent) => void) | null = null

  const applyTheme = () => {
    document.documentElement.classList.toggle('dark', isDark.value)
  }

  watch(isDark, applyTheme)

  const loadTheme = async () => {
    // Clean up previous media query listener
    if (mediaQueryHandler) {
      window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', mediaQueryHandler)
      mediaQueryHandler = null
    }

    try {
      const res = await getSettings()
      const theme = res.data.data?.['ui.theme'] || 'light'
      if (theme === 'dark') {
        isDark.value = true
      } else if (theme === 'auto') {
        isDark.value = window.matchMedia('(prefers-color-scheme: dark)').matches
        mediaQueryHandler = (e: MediaQueryListEvent) => {
          isDark.value = e.matches
        }
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', mediaQueryHandler)
      } else {
        isDark.value = false
      }
      applyTheme()
    } catch {
      // fallback to light
      isDark.value = false
      applyTheme()
    }
  }

  // Clean up on unmount
  onUnmounted(() => {
    if (mediaQueryHandler) {
      window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', mediaQueryHandler)
      mediaQueryHandler = null
    }
  })

  return { isDark, loadTheme }
}
