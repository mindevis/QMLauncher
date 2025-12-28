// Global types for QMLauncher application

// Base component props
export interface BaseComponentProps {
  className?: string
  children?: React.ReactNode
}

// Theme types
export type Theme = 'light' | 'dark' | 'system'

// Button variants (from shadcn/ui)
export type ButtonVariant =
  | 'default'
  | 'destructive'
  | 'outline'
  | 'secondary'
  | 'ghost'
  | 'link'

export type ButtonSize = 'default' | 'sm' | 'lg' | 'icon'

// Card component types
export interface CardProps extends BaseComponentProps {
  variant?: 'default' | 'elevated' | 'outlined'
}

// Application state types
export interface AppState {
  theme: Theme
  isLoading: boolean
  error?: string | null
}

// Wails runtime types
export interface WailsRuntime {
  Window: {
    Show(): void
    Hide(): void
    Close(): void
    SetTitle(title: string): void
    SetSize(width: number, height: number): void
    GetSize(): Promise<{ width: number; height: number }>
  }
  Events: {
    On(event: string, callback: (...args: any[]) => void): void
    Off(event: string, callback?: (...args: any[]) => void): void
    Emit(event: string, ...args: any[]): void
  }
  Log: {
    Info(message: string): void
    Debug(message: string): void
    Error(message: string): void
  }
}

// Go backend bindings types
export interface BackendBindings {
  // Add your Go backend function signatures here
  // Example:
  // GetVersion(): Promise<string>
  // SaveConfig(config: AppConfig): Promise<boolean>
}

// Application configuration
export interface AppConfig {
  theme: Theme
  window: {
    width: number
    height: number
    center: boolean
  }
  features: {
    autoUpdate: boolean
    telemetry: boolean
  }
}

// API Response types
export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  error?: string
  message?: string
}

// Loading states
export type LoadingState = 'idle' | 'pending' | 'fulfilled' | 'rejected'

// Generic async state
export interface AsyncState<T> {
  data: T | null
  loading: boolean
  error: string | null
}
