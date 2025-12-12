/**
 * Window dragging utility for Wails frameless windows
 * Optimized with requestAnimationFrame for smooth dragging
 */

import { WindowGetPosition, WindowSetPosition } from '../../wailsjs/runtime/runtime'

let isDragging = false
let dragStartX = 0
let dragStartY = 0
let windowStartX = 0
let windowStartY = 0
let animationFrameId: number | null = null
let pendingX = 0
let pendingY = 0
let hasPendingUpdate = false

export function setupWindowDrag(element: HTMLElement) {
  const updatePosition = () => {
    if (!isDragging || !hasPendingUpdate) {
      animationFrameId = null
      return
    }
    
    WindowSetPosition(pendingX, pendingY)
    hasPendingUpdate = false
    animationFrameId = null
  }
  
  const handleMouseDown = async (e: MouseEvent) => {
    if (e.button !== 0) return // Only handle left mouse button
    
    // Используем screenX/screenY для координат относительно экрана
    dragStartX = e.screenX
    dragStartY = e.screenY
    
    try {
      const position = await WindowGetPosition()
      windowStartX = position.x
      windowStartY = position.y
      isDragging = true
    } catch (error) {
      console.error('Error getting window position:', error)
      isDragging = false
      return
    }
    
    element.style.cursor = 'grabbing'
    e.preventDefault()
  }
  
  const handleMouseMove = (e: MouseEvent) => {
    if (!isDragging) return
    
    // Используем screenX/screenY для координат относительно экрана
    // Вычисляем новую позицию окна на основе смещения мыши от начальной точки
    const deltaX = e.screenX - dragStartX
    const deltaY = e.screenY - dragStartY
    
    pendingX = windowStartX + deltaX
    pendingY = windowStartY + deltaY
    hasPendingUpdate = true
    
    // Используем requestAnimationFrame для плавного обновления
    if (animationFrameId === null) {
      animationFrameId = requestAnimationFrame(updatePosition)
    }
  }
  
  const handleMouseUp = () => {
    if (!isDragging) return
    
    // Применяем финальную позицию, если есть ожидающее обновление
    if (hasPendingUpdate) {
      if (animationFrameId === null) {
        WindowSetPosition(pendingX, pendingY)
      } else {
        // Отменяем текущий кадр и применяем финальную позицию сразу
        cancelAnimationFrame(animationFrameId)
        WindowSetPosition(pendingX, pendingY)
        animationFrameId = null
      }
    }
    
    isDragging = false
    hasPendingUpdate = false
    element.style.cursor = ''
  }
  
  element.addEventListener('mousedown', handleMouseDown)
  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', handleMouseUp)
  
  // Cleanup function
  return () => {
    element.removeEventListener('mousedown', handleMouseDown)
    document.removeEventListener('mousemove', handleMouseMove)
    document.removeEventListener('mouseup', handleMouseUp)
    
    if (animationFrameId !== null) {
      cancelAnimationFrame(animationFrameId)
      animationFrameId = null
    }
  }
}

