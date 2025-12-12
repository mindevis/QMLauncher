import { createContext, useContext, useState, useEffect, useCallback, useMemo, ReactNode } from 'react'
import { wailsAPI } from '../bridge'

export type Language = 'en' | 'ru'

interface I18nContextType {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: string, params?: Record<string, string | number>) => string
}

const I18nContext = createContext<I18nContextType | undefined>(undefined)

export function I18nProvider({ children }: { children: ReactNode }) {
  // Initialize language from localStorage synchronously
  const [language, setLanguageState] = useState<Language>(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem('qmlauncher_language')
      if (stored === 'en' || stored === 'ru') {
        return stored
      }
    }
    return 'ru'
  })
  
  // Load language from settings on mount (async)
  useEffect(() => {
    const loadLanguageFromSettings = async () => {
      try {
        const settings = await wailsAPI.getSettings()
        // Settings doesn't have language field yet, so we'll use localStorage
        // In future, we can add language to Settings struct in Go
      } catch (error) {
        console.error('Error loading language from settings:', error)
      }
    }
    
    loadLanguageFromSettings()
  }, [])

  const [translations, setTranslations] = useState<Record<string, any>>({})

  // Load translations
  useEffect(() => {
    import(`../locales/${language}.json`)
      .then((module) => {
        setTranslations(module.default)
      })
      .catch((error) => {
        console.error(`Failed to load translations for ${language}:`, error)
        // Fallback to English if current language fails
        if (language !== 'en') {
          import('../locales/en.json')
            .then((module) => setTranslations(module.default))
            .catch(() => setTranslations({}))
        }
      })
  }, [language])

  const setLanguage = useCallback(async (lang: Language) => {
    setLanguageState(lang)
    localStorage.setItem('qmlauncher_language', lang)
    
    // Save to settings (when language field is added to Settings)
    // For now, just save to localStorage
  }, [])

  const t = useCallback(
    (key: string, params?: Record<string, string | number>): string => {
      const keys = key.split('.')
      let value: any = translations

      for (const k of keys) {
        if (value && typeof value === 'object' && k in value) {
          value = value[k]
        } else {
          // Return key if translation not found
          return key
        }
      }

      if (typeof value !== 'string') {
        return key
      }

      // Replace parameters in string
      if (params) {
        return value.replace(/\{(\w+)\}/g, (match, paramKey) => {
          return params[paramKey]?.toString() || match
        })
      }

      return value
    },
    [translations]
  )

  const value = useMemo(
    () => ({
      language,
      setLanguage,
      t,
    }),
    [language, setLanguage, t]
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const context = useContext(I18nContext)
  if (context === undefined) {
    throw new Error('useI18n must be used within an I18nProvider')
  }
  return context
}

