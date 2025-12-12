import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

console.log('[Frontend] Starting React application...')

const rootElement = document.getElementById('root')
if (!rootElement) {
  console.error('[Frontend] Root element not found!')
  throw new Error('Root element not found')
}

console.log('[Frontend] Root element found, rendering App...')

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)

console.log('[Frontend] App rendered')

