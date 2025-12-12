import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import { wailsAPI } from '../bridge'

export interface ThemeColors {
  background: string
  backgroundSecondary: string
  backgroundTertiary: string
  foreground: string
  foregroundSecondary: string
  border: string
  borderSecondary: string
  primary: string
  primaryHover: string
  secondary: string
  secondaryHover: string
  accent: string
  accentHover: string
  text: string
  textSecondary: string
  textMuted: string
  success: string
  error: string
  warning: string
  info: string
}

export interface Theme {
  id: string
  name: string
  displayName: string
  colors: ThemeColors
  isCustom?: boolean
}

const defaultThemes: Theme[] = [
  {
    id: 'dark',
    name: 'dark',
    displayName: 'Темная',
    colors: {
      background: '#252525',
      backgroundSecondary: '#343434',
      backgroundTertiary: '#444444',
      foreground: '#fafafa',
      foregroundSecondary: '#f5f5f5',
      border: 'rgba(255, 255, 255, 0.1)',
      borderSecondary: 'rgba(255, 255, 255, 0.15)',
      primary: '#ebebeb',
      primaryHover: '#d0d0d0',
      secondary: '#444444',
      secondaryHover: '#555555',
      accent: '#444444',
      accentHover: '#555555',
      text: '#fafafa',
      textSecondary: '#e0e0e0',
      textMuted: '#b5b5b5',
      success: '#10b981',
      error: '#ef4444',
      warning: '#f59e0b',
      info: '#3b82f6'
    }
  },
  {
    id: 'light',
    name: 'light',
    displayName: 'Светлая',
    colors: {
      background: '#ffffff',
      backgroundSecondary: '#f7f7f7',
      backgroundTertiary: '#f5f5f5',
      foreground: '#252525',
      foregroundSecondary: '#1a1a1a',
      border: '#ebebeb',
      borderSecondary: '#d0d0d0',
      primary: '#343434',
      primaryHover: '#252525',
      secondary: '#f7f7f7',
      secondaryHover: '#e8e8e8',
      accent: '#f7f7f7',
      accentHover: '#e8e8e8',
      text: '#252525',
      textSecondary: '#404040',
      textMuted: '#8e8e8e',
      success: '#10b981',
      error: '#ef4444',
      warning: '#f59e0b',
      info: '#3b82f6'
    }
  }
]

interface ThemeContextType {
  currentTheme: Theme
  themes: Theme[]
  setTheme: (themeId: string) => void
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themes] = useState<Theme[]>(defaultThemes)
  const [currentTheme, setCurrentTheme] = useState<Theme>(defaultThemes[0])

  // Load theme from storage on mount
  useEffect(() => {
    loadThemeFromStorage()
  }, [])

  // Apply theme to CSS variables
  useEffect(() => {
    applyTheme(currentTheme)
    saveThemeToStorage(currentTheme.id)
  }, [currentTheme])

  const loadThemeFromStorage = async () => {
    try {
      const stored = localStorage.getItem('qmlauncher_theme')
      if (stored) {
        const theme = themes.find(t => t.id === stored)
        if (theme) {
          setCurrentTheme(theme)
          return
        }
      }
    } catch (error) {
      console.error('Error loading theme from storage:', error)
    }
  }

  const saveThemeToStorage = async (themeId: string) => {
    try {
      localStorage.setItem('qmlauncher_theme', themeId)
      // In future, we can save to Settings when theme field is added
    } catch (error) {
      console.error('Error saving theme to storage:', error)
    }
  }

  const applyTheme = (theme: Theme) => {
    const root = document.documentElement
    
    // Просто переключаем класс dark, CSS переменные уже определены в index.css
    if (theme.id === 'dark') {
      root.classList.add('dark')
    } else {
      root.classList.remove('dark')
    }
  }

  const setTheme = (themeId: string) => {
    const theme = themes.find(t => t.id === themeId)
    if (theme) {
      setCurrentTheme(theme)
    }
  }

  return (
    <ThemeContext.Provider
      value={{
        currentTheme,
        themes,
        setTheme
      }}
    >
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}

