import crypto from 'crypto'
import os from 'os'
import { app } from 'electron'

// Encryption key derivation from HWID or app-specific data
// This ensures the key is unique per installation but consistent
function getEncryptionKey(): Buffer {
  // Use a combination of machine-specific identifiers
  const machineId = os.hostname()
  
  // Create a consistent key from machine ID
  const keyMaterial = `QMLauncher-${machineId}-${app.getName()}`
  return crypto.createHash('sha256').update(keyMaterial).digest()
}

const ALGORITHM = 'aes-256-gcm'
const IV_LENGTH = 16
const SALT_LENGTH = 64
const TAG_LENGTH = 16

/**
 * Encrypt data using AES-256-GCM
 */
export function encrypt(data: string): string {
  try {
    const key = getEncryptionKey()
    const iv = crypto.randomBytes(IV_LENGTH)
    const salt = crypto.randomBytes(SALT_LENGTH)
    
    // Derive key from master key and salt
    const derivedKey = crypto.pbkdf2Sync(key, salt, 100000, 32, 'sha256')
    
    const cipher = crypto.createCipheriv(ALGORITHM, derivedKey, iv)
    
    let encrypted = cipher.update(data, 'utf8')
    encrypted = Buffer.concat([encrypted, cipher.final()])
    
    const tag = cipher.getAuthTag()
    
    // Combine salt + iv + tag + encrypted data
    const result = Buffer.concat([
      salt,
      iv,
      tag,
      encrypted
    ])
    
    return result.toString('base64')
  } catch (error) {
    console.error('Encryption error:', error)
    throw error
  }
}

/**
 * Decrypt data using AES-256-GCM
 */
export function decrypt(encryptedData: string): string {
  try {
    const key = getEncryptionKey()
    const data = Buffer.from(encryptedData, 'base64')
    
    // Extract components
    const salt = data.slice(0, SALT_LENGTH)
    const iv = data.slice(SALT_LENGTH, SALT_LENGTH + IV_LENGTH)
    const tag = data.slice(SALT_LENGTH + IV_LENGTH, SALT_LENGTH + IV_LENGTH + TAG_LENGTH)
    const encrypted = data.slice(SALT_LENGTH + IV_LENGTH + TAG_LENGTH)
    
    // Derive key from master key and salt
    const derivedKey = crypto.pbkdf2Sync(key, salt, 100000, 32, 'sha256')
    
    const decipher = crypto.createDecipheriv(ALGORITHM, derivedKey, iv)
    decipher.setAuthTag(tag)
    
    let decrypted = decipher.update(encrypted)
    decrypted = Buffer.concat([decrypted, decipher.final()])
    
    return decrypted.toString('utf8')
  } catch (error) {
    console.error('Decryption error:', error)
    throw error
  }
}

