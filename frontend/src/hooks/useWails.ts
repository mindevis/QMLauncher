import { useEffect, useState, useCallback } from 'react'
import type { AsyncState, WailsRuntime, BackendBindings } from '@/types'

// Hook for accessing Wails runtime
export function useWailsRuntime(): WailsRuntime | null {
  const [runtime, setRuntime] = useState<WailsRuntime | null>(null)

  useEffect(() => {
    if (typeof window !== 'undefined' && window.runtime) {
      setRuntime(window.runtime)
    }
  }, [])

  return runtime
}

// Hook for accessing backend bindings
export function useBackend(): BackendBindings | null {
  const [backend, setBackend] = useState<BackendBindings | null>(null)

  useEffect(() => {
    if (typeof window !== 'undefined' && window.go?.main?.App) {
      setBackend(window.go.main.App)
    }
  }, [])

  return backend
}

// Generic hook for backend calls
export function useBackendCall<TArgs extends any[], TResult>(
  method: (...args: TArgs) => Promise<TResult>
) {
  const [state, setState] = useState<AsyncState<TResult>>({
    data: null,
    loading: false,
    error: null,
  })

  const execute = useCallback(async (...args: TArgs) => {
    setState({ data: null, loading: true, error: null })

    try {
      const result = await method(...args)
      setState({ data: result, loading: false, error: null })
      return result
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      setState({ data: null, loading: false, error: errorMessage })
      throw error
    }
  }, [method])

  return {
    ...state,
    execute,
    reset: () => setState({ data: null, loading: false, error: null }),
  }
}

// Hook for Wails events
export function useWailsEvent<T = any>(
  eventName: string,
  callback: (data: T) => void
) {
  const runtime = useWailsRuntime()

  useEffect(() => {
    if (runtime && eventName && callback) {
      runtime.Events.On(eventName, callback)

      return () => {
        runtime.Events.Off(eventName, callback)
      }
    }
    return undefined
  }, [runtime, eventName, callback])
}

// Export types for convenience
export type { WailsRuntime, BackendBindings } from '@/types'

// Hook for window management
export function useWindow() {
  const runtime = useWailsRuntime()

  const setTitle = useCallback((title: string) => {
    runtime?.Window.SetTitle(title)
  }, [runtime])

  const setSize = useCallback((width: number, height: number) => {
    runtime?.Window.SetSize(width, height)
  }, [runtime])

  const getSize = useCallback(async (): Promise<{ width: number; height: number } | null> => {
    if (runtime) {
      return await runtime.Window.GetSize()
    }
    return null
  }, [runtime])

  const show = useCallback(() => {
    runtime?.Window.Show()
  }, [runtime])

  const hide = useCallback(() => {
    runtime?.Window.Hide()
  }, [runtime])

  const close = useCallback(() => {
    runtime?.Window.Close()
  }, [runtime])

  return {
    setTitle,
    setSize,
    getSize,
    show,
    hide,
    close,
  }
}
