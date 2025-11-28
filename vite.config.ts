import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'
import { readFileSync } from 'fs'

// Conditionally import visualizer
let visualizerPlugin: any = null;
if (process.env.ANALYZE === 'true') {
  try {
    const { visualizer } = require('vite-bundle-visualizer');
    visualizerPlugin = visualizer({
      open: false,
      filename: 'dist/stats.html',
      gzipSize: true,
      brotliSize: true,
    });
  } catch (e) {
    // visualizer not available, skip
  }
}

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
  plugins: [
    react(), 
    tailwindcss(),
    ...(visualizerPlugin ? [visualizerPlugin] : []),
  ],
  base: './',
  server: {
    port: 5175,
    strictPort: true
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    // Production optimizations
    minify: 'esbuild', // Fast and efficient minification
    cssMinify: true, // Minify CSS
    sourcemap: process.env.NODE_ENV === 'development', // Source maps only in development
    // Optimize chunk size
    chunkSizeWarningLimit: 1000,
    // Improve build performance
    target: 'esnext',
    rollupOptions: {
      output: {
        // Optimize chunk splitting
        manualChunks: (id) => {
          // Vendor chunks for better caching
          if (id.includes('node_modules')) {
            // React core libraries
            if (id.includes('react') || id.includes('react-dom')) {
              return 'react-vendor';
            }
            // UI libraries
            if (id.includes('@radix-ui')) {
              return 'ui-vendor';
            }
            // Icons
            if (id.includes('lucide-react')) {
              return 'icons-vendor';
            }
            // Electron specific
            if (id.includes('electron-updater')) {
              return 'electron-vendor';
            }
            // Animation
            if (id.includes('framer-motion')) {
              return 'animation-vendor';
            }
            // Other vendor libraries
            return 'vendor';
          }
        },
        // Optimize chunk file names
        chunkFileNames: 'assets/js/[name]-[hash].js',
        entryFileNames: 'assets/js/[name]-[hash].js',
        assetFileNames: (assetInfo) => {
          const info = assetInfo.name?.split('.') || [];
          const ext = info[info.length - 1];
          if (/png|jpe?g|svg|gif|tiff|bmp|ico/i.test(ext)) {
            return `assets/images/[name]-[hash][extname]`;
          }
          if (/woff2?|eot|ttf|otf/i.test(ext)) {
            return `assets/fonts/[name]-[hash][extname]`;
          }
          return `assets/[ext]/[name]-[hash][extname]`;
        },
      },
    },
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

