import { useState, useEffect } from 'react'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { AlertTriangle, Download, X } from 'lucide-react'

// Update information interface
interface UpdateInfo {
  available: boolean
  latestVersion: string
  currentVersion: string
  changelog: string
  downloadSize: string
  releaseUrl: string
}

// Component props
interface UpdateNotificationProps {
  onClose: () => void
  onUpdate: () => void
  updateInfo: UpdateInfo | null
  isDownloading: boolean
  downloadProgress: number
}

export function UpdateNotification({
  onClose,
  onUpdate,
  updateInfo,
  isDownloading,
  downloadProgress
}: UpdateNotificationProps) {
  if (!updateInfo?.available) {
    return null
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 max-w-md">
      <Card className="shadow-lg border-l-4 border-l-blue-500">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-blue-500" />
              <CardTitle className="text-lg">Update Available</CardTitle>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={onClose}
              className="h-8 w-8 p-0"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
          <CardDescription>
            Version {updateInfo.latestVersion} is available (current: {updateInfo.currentVersion})
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">
          {updateInfo.changelog && (
            <div>
              <h4 className="font-semibold mb-2">What's New:</h4>
              <div className="text-sm text-muted-foreground max-h-24 overflow-y-auto">
                {updateInfo.changelog}
              </div>
            </div>
          )}

          {updateInfo.downloadSize && (
            <div className="flex items-center gap-2">
              <Download className="h-4 w-4" />
              <span className="text-sm">Download size: {updateInfo.downloadSize}</span>
            </div>
          )}

          {isDownloading && (
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span>Downloading...</span>
                <span>{Math.round(downloadProgress)}%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${downloadProgress}%` }}
                />
              </div>
            </div>
          )}

          <div className="flex gap-2">
            <Button
              onClick={onUpdate}
              disabled={isDownloading}
              className="flex-1"
            >
              {isDownloading ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2" />
                  Downloading...
                </>
              ) : (
                <>
                  <Download className="h-4 w-4 mr-2" />
                  Update Now
                </>
              )}
            </Button>

            <Button
              variant="outline"
              onClick={() => window.open(updateInfo.releaseUrl, '_blank')}
              disabled={isDownloading}
            >
              View Release
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// Hook for managing updates
export function useUpdates() {
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null)
  const [isDownloading, setIsDownloading] = useState(false)
  const [downloadProgress, setDownloadProgress] = useState(0)
  const [isVisible, setIsVisible] = useState(false)

  // Check for updates on mount
  useEffect(() => {
    checkForUpdates()
  }, [])

  const checkForUpdates = async () => {
    try {
      // This would call the Wails backend
      // For now, we'll simulate an update check
      const mockUpdate: UpdateInfo = {
        available: true,
        latestVersion: "1.1.0",
        currentVersion: "1.0.0",
        changelog: "• Added smart instance import with merge mode\n• Enhanced cross-platform path normalization\n• Improved error messages and user guidance",
        downloadSize: "15.2 MB",
        releaseUrl: "https://github.com/qdevis/QMLauncher/releases/tag/v1.1.0"
      }

      setUpdateInfo(mockUpdate)
      setIsVisible(true)
    } catch (error) {
      console.error('Failed to check for updates:', error)
    }
  }

  const startUpdate = async () => {
    if (!updateInfo) return

    setIsDownloading(true)
    setDownloadProgress(0)

    try {
      // Simulate download progress
      const interval = setInterval(() => {
        setDownloadProgress(prev => {
          if (prev >= 100) {
            clearInterval(interval)
            // Simulate successful update
            setTimeout(() => {
              setIsDownloading(false)
              setIsVisible(false)
              // In real app, would restart the application
              alert('Update completed! Application will restart.')
            }, 1000)
            return 100
          }
          return prev + Math.random() * 15
        })
      }, 500)
    } catch (error) {
      console.error('Failed to download update:', error)
      setIsDownloading(false)
    }
  }

  const dismiss = () => {
    setIsVisible(false)
  }

  return {
    updateInfo,
    isDownloading,
    downloadProgress,
    isVisible,
    checkForUpdates,
    startUpdate,
    dismiss
  }
}