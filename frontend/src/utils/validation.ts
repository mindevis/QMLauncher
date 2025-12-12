/**
 * Валидация с использованием Zod
 */
import { z } from 'zod'

// Схема для логина
export const loginSchema = z.object({
  login: z.string().min(1, 'Логин обязателен').max(100, 'Логин слишком длинный'),
  password: z.string().min(1, 'Пароль обязателен').min(6, 'Пароль должен быть не менее 6 символов'),
})

export type LoginFormData = z.infer<typeof loginSchema>

// Схема для сервера
export const serverSchema = z.object({
  id: z.number().positive(),
  name: z.string().min(1).max(100),
  server_name: z.string().optional(),
  server_address: z.string().optional(),
  server_port: z.number().int().min(1).max(65535).optional(),
  minecraft_version: z.string().optional(),
  description: z.string().optional(),
  preview_image_url: z.string().url().optional().or(z.literal('')),
  server_uuid: z.string().optional(),
  server_status: z.enum(['online', 'offline', 'unknown']).optional(),
  loader_enabled: z.boolean().optional(),
  loader_type: z.string().optional(),
})

export type ServerFormData = z.infer<typeof serverSchema>

// Схема для настроек сервера
export const serverSettingsSchema = z.object({
  minecraftPath: z.string().min(1, 'Путь к Minecraft обязателен'),
  javaPath: z.string().min(1, 'Путь к Java обязателен'),
  jvmArgs: z.string().optional(),
  windowWidth: z.number().int().min(800).max(7680).optional(),
  windowHeight: z.number().int().min(600).max(4320).optional(),
  resolution: z.string().optional(),
  customResolution: z.string().optional(),
  minMemory: z.number().int().min(512).max(16384).optional(),
})

export type ServerSettingsFormData = z.infer<typeof serverSettingsSchema>

