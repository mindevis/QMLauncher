/**
 * API Client for QMLauncher
 * Uses Wails Go backend for HTTP requests (for CORS handling)
 */

import { API_BASE_URL } from '../config/api'
import { wailsAPI } from '../bridge'

interface ApiRequestOptions extends RequestInit {
  authToken?: string | null
}

interface ApiResponse<T = any> {
  ok: boolean
  status: number
  statusText: string
  data: T
  headers: Record<string, string>
}

/**
 * Makes an API request using Wails Go backend (for CORS handling)
 * Includes retry logic for network errors
 */
export async function apiRequest<T = any>(
  endpoint: string,
  options: ApiRequestOptions = {},
  retries = 3
): Promise<ApiResponse<T>> {
  const { authToken, ...fetchOptions } = options
  const url = endpoint.startsWith('http') ? endpoint : `${API_BASE_URL}${endpoint}`

  // Prepare headers
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(fetchOptions.headers as Record<string, string>),
  }

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`
  }

  // Use Wails API for requests with retry logic
  for (let attempt = 0; attempt < retries; attempt++) {
    try {
      const method = (fetchOptions.method || 'GET').toUpperCase()
      const body = fetchOptions.body ? (typeof fetchOptions.body === 'string' ? fetchOptions.body : JSON.stringify(fetchOptions.body)) : undefined
      
      const response = await wailsAPI.apiRequest(url, method, headers, body)
      
      return {
        ok: response.status >= 200 && response.status < 300,
        status: response.status || 200,
        statusText: response.statusText || 'OK',
        data: response.data as T,
        headers: response.headers || {},
      }
    } catch (error) {
      const isLastAttempt = attempt === retries - 1
      const isNetworkError = error instanceof Error && (
        error.message.includes('network') || 
        error.message.includes('timeout') ||
        error.message.includes('ECONNREFUSED')
      )
      
      if (isLastAttempt || !isNetworkError) {
        console.error('[API Client] Request failed:', error)
        throw error
      }
      
      // Exponential backoff: 1s, 2s, 4s
      await new Promise(resolve => setTimeout(resolve, 1000 * Math.pow(2, attempt)))
    }
  }
  
  throw new Error('Request failed after retries')
}

/**
 * Convenience methods for common HTTP verbs
 */
export const api = {
  get: <T = any>(endpoint: string, options?: ApiRequestOptions) =>
    apiRequest<T>(endpoint, { ...options, method: 'GET' }),

  post: <T = any>(endpoint: string, body?: any, options?: ApiRequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'POST',
      body: JSON.stringify(body),
    }),

  put: <T = any>(endpoint: string, body?: any, options?: ApiRequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(body),
    }),

  patch: <T = any>(endpoint: string, body?: any, options?: ApiRequestOptions) =>
    apiRequest<T>(endpoint, {
      ...options,
      method: 'PATCH',
      body: JSON.stringify(body),
    }),

  delete: <T = any>(endpoint: string, options?: ApiRequestOptions) =>
    apiRequest<T>(endpoint, { ...options, method: 'DELETE' }),
}

