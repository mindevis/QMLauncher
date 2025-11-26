// API configuration - values are replaced at build time by Vite
// These are TypeScript declarations for the build-time constants

declare const __QM_SERVER_API_BASE_URL__: string | undefined
declare const __QM_SERVER_API_HOST__: string | undefined
declare const __QM_SERVER_API_PORT__: string | undefined
declare const __QM_SERVER_API_PROTOCOL__: string | undefined
declare const __QM_SERVER_API_BASE_PATH__: string | undefined

// Export API configuration
export const API_CONFIG = {
  BASE_URL: typeof __QM_SERVER_API_BASE_URL__ !== 'undefined' 
    ? __QM_SERVER_API_BASE_URL__ 
    : (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8000/api/v1'),
  HOST: typeof __QM_SERVER_API_HOST__ !== 'undefined' 
    ? __QM_SERVER_API_HOST__ 
    : (import.meta.env.VITE_API_HOST || 'localhost'),
  PORT: typeof __QM_SERVER_API_PORT__ !== 'undefined' 
    ? __QM_SERVER_API_PORT__ 
    : (import.meta.env.VITE_API_PORT || '8000'),
  PROTOCOL: typeof __QM_SERVER_API_PROTOCOL__ !== 'undefined' 
    ? __QM_SERVER_API_PROTOCOL__ 
    : (import.meta.env.VITE_API_PROTOCOL || 'http'),
  BASE_PATH: typeof __QM_SERVER_API_BASE_PATH__ !== 'undefined' 
    ? __QM_SERVER_API_BASE_PATH__ 
    : (import.meta.env.VITE_API_BASE_PATH || '/api/v1')
}

// Convenience export for the full API base URL
export const API_BASE_URL = API_CONFIG.BASE_URL

