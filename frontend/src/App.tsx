import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { AppProvider, useApp } from '@/contexts/AppContext'
import { useWindow } from '@/hooks/useWails'
import { UpdateNotification, useUpdates } from '@/components/UpdateNotification'

// TypeScript interface for tech stack item
interface TechStackItem {
  icon: string
  title: string
  description: string
}

// TypeScript interface for feature item
interface FeatureItem {
  text: string
  checked: boolean
}

// Main App component (wrapped with context)
function AppContent() {
  const { theme, setTheme } = useApp()
  const { setTitle } = useWindow()
  const updateState = useUpdates()

  // Typed data structures
  const techStack: TechStackItem[] = [
    {
      icon: '⚛️',
      title: 'React + Vite',
      description: 'Modern frontend framework with TypeScript'
    },
    {
      icon: '🎨',
      title: 'shadcn/ui',
      description: 'Beautiful UI components with full TypeScript support'
    },
    {
      icon: '🔧',
      title: 'Wails + Go',
      description: 'Desktop application backend'
    }
  ]

  const features: FeatureItem[] = [
    { text: 'Cross-platform desktop application', checked: true },
    { text: 'Modern React frontend with Vite', checked: true },
    { text: 'Go backend with Wails', checked: true },
    { text: 'shadcn/ui components with TypeScript', checked: true },
    { text: 'Tailwind CSS styling', checked: true },
    { text: 'Hot reload development', checked: true },
    { text: 'Full TypeScript support', checked: true },
    { text: 'Fast build times', checked: true },
  ]

  // TypeScript event handlers
  const handleThemeChange = (newTheme: 'light' | 'dark' | 'system') => {
    setTheme(newTheme)
  }

  const handleWindowTitleChange = () => {
    setTitle('QMLauncher - TypeScript Powered!')
  }

  return (
    <div className="min-h-screen bg-background p-8">
      <div className="max-w-4xl mx-auto space-y-8">
        {/* Header */}
        <div className="text-center space-y-4">
          <div className="w-32 h-32 mx-auto bg-primary/10 rounded-full flex items-center justify-center">
            <span className="text-4xl">🚀</span>
          </div>
          <h1 className="text-4xl font-bold text-foreground">QMLauncher</h1>
          <p className="text-lg text-muted-foreground">
            Modern desktop application built with Wails, Go and React + TypeScript
          </p>
        </div>

        {/* Theme Controls */}
        <div className="flex justify-center gap-2">
          {(['light', 'dark', 'system'] as const).map((themeOption) => (
            <Button
              key={themeOption}
              variant={theme === themeOption ? 'default' : 'outline'}
              size="sm"
              onClick={() => handleThemeChange(themeOption)}
            >
              {themeOption.charAt(0).toUpperCase() + themeOption.slice(1)}
            </Button>
          ))}
        </div>

        {/* Main content */}
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Welcome to QMLauncher</CardTitle>
              <CardDescription>
                This is a demonstration of shadcn/ui components with full TypeScript support integrated with Wails.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <Button
                  className="w-full"
                  onClick={handleWindowTitleChange}
                >
                  Update Window Title
                </Button>
                <Button variant="outline" className="w-full">
                  Learn More About TypeScript
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Features</CardTitle>
              <CardDescription>
                What makes QMLauncher special with TypeScript
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2 text-sm text-muted-foreground">
                {features.map((feature, index) => (
                  <li key={index}>
                    {feature.checked ? '✅' : '❌'} {feature.text}
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </div>

        {/* Tech Stack */}
        <Card>
          <CardHeader>
            <CardTitle>Tech Stack</CardTitle>
            <CardDescription>
              Technologies used in this TypeScript-powered project
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              {techStack.map((tech, index) => (
                <div key={index} className="text-center">
                  <div className="text-2xl mb-2">{tech.icon}</div>
                  <h3 className="font-semibold">{tech.title}</h3>
                  <p className="text-sm text-muted-foreground">{tech.description}</p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* TypeScript Info */}
        <Card>
          <CardHeader>
            <CardTitle>TypeScript Benefits</CardTitle>
            <CardDescription>
              Why TypeScript makes our application better
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 text-sm">
              <div>
                <h4 className="font-semibold mb-2">🛡️ Type Safety</h4>
                <p className="text-muted-foreground">
                  Catch errors at compile time, not runtime. Full IntelliSense support.
                </p>
              </div>
              <div>
                <h4 className="font-semibold mb-2">📚 Better DX</h4>
                <p className="text-muted-foreground">
                  Enhanced developer experience with autocompletion and refactoring tools.
                </p>
              </div>
              <div>
                <h4 className="font-semibold mb-2">🔧 Maintainability</h4>
                <p className="text-muted-foreground">
                  Self-documenting code with interfaces and type definitions.
                </p>
              </div>
              <div>
                <h4 className="font-semibold mb-2">🚀 Performance</h4>
                <p className="text-muted-foreground">
                  Better optimization and tree-shaking with static type checking.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Update Notification */}
        {updateState.isVisible && (
          <UpdateNotification
            updateInfo={updateState.updateInfo}
            isDownloading={updateState.isDownloading}
            downloadProgress={updateState.downloadProgress}
            onClose={updateState.dismiss}
            onUpdate={updateState.startUpdate}
          />
        )}
      </div>
    </div>
  )
}

// Main App component with providers
function App() {
  return (
    <AppProvider>
      <AppContent />
    </AppProvider>
  )
}

export default App
