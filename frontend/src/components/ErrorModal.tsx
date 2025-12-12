import React from 'react'
import { AlertCircle, X } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { Button } from './ui/button'
import { createPortal } from 'react-dom'
import { useI18n } from '../contexts/I18nContext'

interface ErrorModalProps {
  isOpen: boolean
  error: string | null
  onClose: () => void
  onRetry?: () => void
}

export function ErrorModal({ isOpen, error, onClose, onRetry }: ErrorModalProps) {
  const { t } = useI18n()
  const [mounted, setMounted] = React.useState(false)
  
  React.useEffect(() => {
    setMounted(true)
  }, [])
  
  if (!mounted || !isOpen || !error) {
    return null
  }
  
  const modalContent = (
    <AnimatePresence>
      {isOpen && error && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-[9999] bg-black/80 backdrop-blur-sm"
            onClick={onClose}
          />
          
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: -20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: -20 }}
            className="fixed left-1/2 top-1/2 z-[10000] w-full max-w-md -translate-x-1/2 -translate-y-1/2 no-drag"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="rounded-lg border bg-background p-6 shadow-lg">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-12 h-12 rounded-xl flex items-center justify-center bg-destructive/20">
                    <AlertCircle className="w-6 h-6 text-destructive" />
                  </div>
                  <h2 className="text-lg font-semibold text-foreground">
                    {t('error.connectionTitle')}
                  </h2>
                </div>
                <button
                  onClick={onClose}
                  className="rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                >
                  <X className="h-4 w-4" />
                  <span className="sr-only">Close</span>
                </button>
              </div>
              
              <p className="text-sm text-muted-foreground mb-6">
                {error}
              </p>
              
              <div className="flex flex-col-reverse sm:flex-row gap-2 sm:justify-end">
                {onRetry && (
                  <Button onClick={onRetry} className="w-full sm:w-auto">
                    {t('common.retry')}
                  </Button>
                )}
                <Button variant="outline" onClick={onClose} className="w-full sm:w-auto">
                  {t('common.close')}
                </Button>
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
  
  return createPortal(modalContent, document.body)
}

