/**
 * Безопасное хранение токенов
 * Использует простую обфускацию для токенов (в будущем можно добавить шифрование через Go backend)
 */

const TOKEN_KEY = 'qmlauncher_auth_token'
const ENCRYPTION_KEY = 'qmlauncher_encryption_key' // В production должен быть более безопасным

/**
 * Простая обфускация токена (не является криптографически стойким, но лучше чем plain text)
 * В будущем можно заменить на шифрование через Go backend
 */
function obfuscateToken(token: string): string {
  // Простая обфускация: XOR с ключом
  const key = ENCRYPTION_KEY
  let obfuscated = ''
  for (let i = 0; i < token.length; i++) {
    obfuscated += String.fromCharCode(
      token.charCodeAt(i) ^ key.charCodeAt(i % key.length)
    )
  }
  // Base64 для безопасного хранения
  return btoa(obfuscated)
}

function deobfuscateToken(obfuscated: string): string {
  try {
    const decoded = atob(obfuscated)
    const key = ENCRYPTION_KEY
    let token = ''
    for (let i = 0; i < decoded.length; i++) {
      token += String.fromCharCode(
        decoded.charCodeAt(i) ^ key.charCodeAt(i % key.length)
      )
    }
    return token
  } catch (error) {
    throw new Error('Failed to deobfuscate token')
  }
}

/**
 * Сохранить токен с обфускацией
 */
export function saveAuthToken(token: string): void {
  try {
    const obfuscated = obfuscateToken(token)
    localStorage.setItem(TOKEN_KEY, obfuscated)
  } catch (error) {
    console.error('Failed to save auth token:', error)
    // Fallback: сохранить как есть (для совместимости)
    localStorage.setItem(TOKEN_KEY, token)
  }
}

/**
 * Получить токен с деобфускацией
 */
export function getAuthToken(): string | null {
  try {
    const obfuscated = localStorage.getItem(TOKEN_KEY)
    if (!obfuscated) return null

    // Попытка деобфускации
    try {
      return deobfuscateToken(obfuscated)
    } catch {
      // Если не удалось деобфусцировать, возможно это старый формат
      return obfuscated
    }
  } catch (error) {
    console.error('Failed to get auth token:', error)
    return null
  }
}

/**
 * Удалить токен
 */
export function removeAuthToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

/**
 * Проверить наличие токена
 */
export function hasAuthToken(): boolean {
  return localStorage.getItem(TOKEN_KEY) !== null
}

