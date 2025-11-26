// API configuration for Electron main process
// Values are read from environment variables set during build

const getApiConfig = () => {
  // In production, these should be set as environment variables during build
  // In development, fallback to defaults
  const API_HOST = process.env.QM_SERVER_API_HOST || 'localhost'
  const API_PORT = process.env.QM_SERVER_API_PORT || '8000'
  const API_PROTOCOL = process.env.QM_SERVER_API_PROTOCOL || 'http'
  const API_BASE_PATH = process.env.QM_SERVER_API_BASE_PATH || '/api/v1'
  
  const BASE_URL = `${API_PROTOCOL}://${API_HOST}:${API_PORT}${API_BASE_PATH}`
  
  return {
    BASE_URL,
    HOST: API_HOST,
    PORT: API_PORT,
    PROTOCOL: API_PROTOCOL,
    BASE_PATH: API_BASE_PATH
  }
}

export const API_CONFIG = getApiConfig()
export const API_BASE_URL = API_CONFIG.BASE_URL

