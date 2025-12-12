/**
 * Performance monitoring utilities
 */
import { logger } from './logger'

interface PerformanceMetric {
  name: string
  duration: number
  timestamp: number
  metadata?: Record<string, any>
}

class PerformanceMonitor {
  private metrics: PerformanceMetric[] = []
  private maxMetrics = 100 // Хранить последние 100 метрик

  /**
   * Измеряет время выполнения функции
   */
  async measure<T>(
    name: string,
    fn: () => Promise<T>,
    metadata?: Record<string, any>
  ): Promise<T> {
    const start = performance.now()
    try {
      const result = await fn()
      const duration = performance.now() - start
      this.recordMetric(name, duration, metadata)
      return result
    } catch (error) {
      const duration = performance.now() - start
      this.recordMetric(name, duration, { ...metadata, error: true })
      throw error
    }
  }

  /**
   * Синхронное измерение
   */
  measureSync<T>(
    name: string,
    fn: () => T,
    metadata?: Record<string, any>
  ): T {
    const start = performance.now()
    try {
      const result = fn()
      const duration = performance.now() - start
      this.recordMetric(name, duration, metadata)
      return result
    } catch (error) {
      const duration = performance.now() - start
      this.recordMetric(name, duration, { ...metadata, error: true })
      throw error
    }
  }

  /**
   * Записывает метрику
   */
  private recordMetric(
    name: string,
    duration: number,
    metadata?: Record<string, any>
  ) {
    const metric: PerformanceMetric = {
      name,
      duration,
      timestamp: Date.now(),
      metadata,
    }

    this.metrics.push(metric)

    // Ограничиваем размер массива
    if (this.metrics.length > this.maxMetrics) {
      this.metrics.shift()
    }

    // Логируем медленные операции (> 1 секунды)
    if (duration > 1000) {
      logger.warn(`Slow operation detected: ${name} took ${duration.toFixed(2)}ms`, {
        duration,
        metadata,
      })
    }

    // Можно добавить отправку в Go backend
    // wailsAPI.logMetric(name, duration, metadata)
  }

  /**
   * Получить все метрики
   */
  getMetrics(): PerformanceMetric[] {
    return [...this.metrics]
  }

  /**
   * Получить метрики по имени
   */
  getMetricsByName(name: string): PerformanceMetric[] {
    return this.metrics.filter((m) => m.name === name)
  }

  /**
   * Получить среднее время выполнения операции
   */
  getAverageDuration(name: string): number | null {
    const metrics = this.getMetricsByName(name)
    if (metrics.length === 0) return null

    const sum = metrics.reduce((acc, m) => acc + m.duration, 0)
    return sum / metrics.length
  }

  /**
   * Очистить метрики
   */
  clear() {
    this.metrics = []
  }
}

export const performanceMonitor = new PerformanceMonitor()

