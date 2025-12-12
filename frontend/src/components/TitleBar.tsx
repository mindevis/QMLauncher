import { Minus, Square, X, Maximize2 } from 'lucide-react'
import { useState, useEffect, useRef } from 'react'
import { Button } from './ui/button'
import { ThemeSelector } from './ThemeSelector'
import { LanguageSelector } from './LanguageSelector'
import { QMLogo } from './QMLogo'
import { useI18n } from '../contexts/I18nContext'
import { wailsAPI } from '../bridge'
import { setupWindowDrag } from '../utils/window-drag'

export function TitleBar() {
  const { t } = useI18n()
  const [isMaximized, setIsMaximized] = useState(false)
  const dragRegionRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    wailsAPI.windowIsMaximized().then(setIsMaximized)
  }, [])

  useEffect(() => {
    if (dragRegionRef.current) {
      const cleanup = setupWindowDrag(dragRegionRef.current)
      return cleanup
    }
  }, [])

  const handleMinimize = () => {
    wailsAPI.windowMinimize()
  }

  const handleMaximize = () => {
    wailsAPI.windowMaximize()
    setIsMaximized(!isMaximized)
  }

  const handleClose = () => {
    wailsAPI.windowClose()
  }

  return (
    <div className="h-10 bg-card flex items-center justify-between px-4 rounded-t-2xl">
      {/* Left side - logo/title */}
      <div 
        ref={dragRegionRef}
        className="flex items-center gap-2 drag-region flex-1 cursor-grab active:cursor-grabbing"
      >
        <QMLogo className="h-6 w-6 dark:invert flex-shrink-0" />
        <span className="text-sm font-medium text-muted-foreground">
          QMLauncher
        </span>
      </div>
      
      {/* Right side - window controls */}
      <div className="flex items-center gap-1 no-drag">
        <div className="flex items-center">
          <LanguageSelector />
          <ThemeSelector />
        </div>
        <Button
          variant="ghost"
          size="icon"
          onClick={handleMinimize}
          className="h-10 w-10 no-drag"
          title={t('window.minimize')}
        >
          <Minus className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          onClick={handleMaximize}
          className="h-10 w-10 no-drag"
          title={isMaximized ? t('window.restore') : t('window.maximize')}
        >
          {isMaximized ? (
            <Square className="w-3.5 h-3.5" />
          ) : (
            <Maximize2 className="w-4 h-4" />
          )}
        </Button>
        <Button
          variant="ghost"
          size="icon"
          onClick={handleClose}
          className="h-10 w-10 no-drag hover:bg-destructive/20 hover:text-destructive"
          title={t('window.close')}
        >
          <X className="w-4 h-4" />
        </Button>
      </div>
    </div>
  )
}
