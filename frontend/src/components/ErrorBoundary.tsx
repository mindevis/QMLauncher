import React, { Component, ErrorInfo, ReactNode } from 'react'
import { AlertTriangle, RefreshCw, Home } from 'lucide-react'
import { Button } from './ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { wailsAPI } from '../bridge'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
  errorInfo: ErrorInfo | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null
    }
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return {
      hasError: true,
      error
    }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // Логируем ошибку
    console.error('ErrorBoundary caught an error:', error, errorInfo)
    
    // Сохраняем информацию об ошибке
    this.setState({
      error,
      errorInfo
    })

    // Отправляем ошибку в Go backend для логирования (если есть такой метод)
    try {
      const errorMessage = error.message || 'Unknown error'
      const errorStack = error.stack || ''
      const componentStack = errorInfo.componentStack || ''
      
      // Логируем через console для отладки
      console.error('[ErrorBoundary] Error details:', {
        message: errorMessage,
        stack: errorStack,
        componentStack
      })
      
      // Можно добавить отправку в Go backend, если будет реализован метод logError
      // wailsAPI.logError(errorMessage, errorStack, { componentStack })
    } catch (logError) {
      console.error('[ErrorBoundary] Failed to log error:', logError)
    }
  }

  handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null
    })
  }

  handleReload = () => {
    window.location.reload()
  }

  render() {
    if (this.state.hasError) {
      // Если передан кастомный fallback, используем его
      if (this.props.fallback) {
        return this.props.fallback
      }

      // Стандартный UI для ошибки
      return (
        <div className="flex items-center justify-center min-h-screen p-8 bg-background">
          <Card className="w-full max-w-2xl">
            <CardHeader>
              <div className="flex items-center gap-3">
                <AlertTriangle className="w-8 h-8 text-destructive" />
                <CardTitle className="text-2xl">Что-то пошло не так</CardTitle>
              </div>
              <CardDescription>
                Произошла непредвиденная ошибка в приложении
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {this.state.error && (
                <div className="p-4 bg-destructive/10 border border-destructive/20 rounded-lg">
                  <p className="font-mono text-sm text-destructive break-all">
                    {this.state.error.message || 'Неизвестная ошибка'}
                  </p>
                  {process.env.NODE_ENV === 'development' && this.state.error.stack && (
                    <details className="mt-2">
                      <summary className="text-xs text-muted-foreground cursor-pointer">
                        Детали ошибки (только в режиме разработки)
                      </summary>
                      <pre className="mt-2 text-xs text-muted-foreground overflow-auto max-h-48 p-2 bg-muted rounded">
                        {this.state.error.stack}
                      </pre>
                    </details>
                  )}
                </div>
              )}

              <div className="flex gap-3">
                <Button onClick={this.handleReset} variant="default">
                  <RefreshCw className="w-4 h-4 mr-2" />
                  Попробовать снова
                </Button>
                <Button onClick={this.handleReload} variant="outline">
                  <Home className="w-4 h-4 mr-2" />
                  Перезагрузить приложение
                </Button>
              </div>

              {process.env.NODE_ENV === 'development' && this.state.errorInfo && (
                <details className="mt-4">
                  <summary className="text-xs text-muted-foreground cursor-pointer">
                    Информация о компоненте (только в режиме разработки)
                  </summary>
                  <pre className="mt-2 text-xs text-muted-foreground overflow-auto max-h-48 p-2 bg-muted rounded">
                    {this.state.errorInfo.componentStack}
                  </pre>
                </details>
              )}
            </CardContent>
          </Card>
        </div>
      )
    }

    return this.props.children
  }
}

