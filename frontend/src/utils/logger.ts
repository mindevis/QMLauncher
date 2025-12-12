/**
 * Централизованное логирование для приложения
 * Позволяет легко переключаться между console.log и отправкой в Go backend
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error'

interface LogContext {
  [key: string]: any
}

class Logger {
  private isDevelopment = import.meta.env.DEV

  private formatMessage(level: LogLevel, message: string, context?: LogContext): string {
    const timestamp = new Date().toISOString()
    const contextStr = context ? ` ${JSON.stringify(context)}` : ''
    return `[${timestamp}] [${level.toUpperCase()}] ${message}${contextStr}`
  }

  private async logToBackend(level: LogLevel, message: string, error?: Error, context?: LogContext) {
    // Можно добавить отправку в Go backend через wailsAPI, если будет реализован метод
    // try {
    //   await wailsAPI.logError(message, error?.stack, context)
    // } catch (err) {
    //   console.error('Failed to log to backend:', err)
    // }
  }

  debug(message: string, context?: LogContext) {
    if (this.isDevelopment) {
      console.debug(this.formatMessage('debug', message, context))
    }
  }

  info(message: string, context?: LogContext) {
    console.log(this.formatMessage('info', message, context))
  }

  warn(message: string, context?: LogContext) {
    console.warn(this.formatMessage('warn', message, context))
    this.logToBackend('warn', message, undefined, context)
  }

  error(message: string, error?: Error, context?: LogContext) {
    const errorMessage = error ? `${message}: ${error.message}` : message
    console.error(this.formatMessage('error', errorMessage, context), error)
    this.logToBackend('error', message, error, context)
  }
}

export const logger = new Logger()

