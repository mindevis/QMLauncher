// Type definitions for Wails runtime
// This file provides TypeScript declarations for the Wails JavaScript runtime

declare global {
  interface Window {
    runtime: WailsRuntime
    go: {
      main: {
        App: BackendBindings
      }
    }
  }
}

// Re-export types
export type { WailsRuntime, BackendBindings } from './index'
