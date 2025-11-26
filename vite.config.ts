import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'
import { readFileSync } from 'fs'

// Get API configuration from environment variables or use defaults
const API_HOST = process.env.QM_SERVER_API_HOST || process.env.VITE_API_HOST || 'localhost'
const API_PORT = process.env.QM_SERVER_API_PORT || process.env.VITE_API_PORT || '8000'
const API_PROTOCOL = process.env.QM_SERVER_API_PROTOCOL || process.env.VITE_API_PROTOCOL || 'http'
const API_BASE_PATH = process.env.QM_SERVER_API_BASE_PATH || process.env.VITE_API_BASE_PATH || '/api/v1'

// Construct full API URL
const API_BASE_URL = `${API_PROTOCOL}://${API_HOST}:${API_PORT}${API_BASE_PATH}`

// Get launcher version from environment, package.json, or default
let launcherVersion = process.env.QM_LAUNCHER_VERSION
if (!launcherVersion) {
  try {
    const packageJson = JSON.parse(readFileSync(path.resolve(__dirname, 'package.json'), 'utf-8'))
    launcherVersion = packageJson.version || '0.0.0'
  } catch {
    launcherVersion = process.env.npm_package_version || '0.0.0'
  }
}

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: './',
  server: {
    port: 5175,
    strictPort: true
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src/renderer'),
      '@shared': path.resolve(__dirname, './src/shared')
    }
  },
  define: {
    // Inline API configuration at build time
    '__QM_SERVER_API_BASE_URL__': JSON.stringify(API_BASE_URL),
    '__QM_SERVER_API_HOST__': JSON.stringify(API_HOST),
    '__QM_SERVER_API_PORT__': JSON.stringify(API_PORT),
    '__QM_SERVER_API_PROTOCOL__': JSON.stringify(API_PROTOCOL),
    '__QM_SERVER_API_BASE_PATH__': JSON.stringify(API_BASE_PATH),
    // Launcher version from environment or package.json
    '__QM_LAUNCHER_VERSION__': JSON.stringify(launcherVersion)
  }
})

