# TypeScript в QMLauncher Frontend

## Обзор

Frontend приложения QMLauncher полностью типизирован с использованием TypeScript для обеспечения максимальной типобезопасности и лучшего developer experience.

## Структура типов

### Основные типы (`src/types/index.ts`)

- `AppState` - состояние приложения
- `Theme` - темы ('light' | 'dark' | 'system')
- `AppConfig` - конфигурация приложения
- `ApiResponse<T>` - общий тип API ответов
- `AsyncState<T>` - состояние асинхронных операций
- `BackendBindings` - типы для Wails backend
- `WailsRuntime` - типы Wails runtime

### Wails интеграция (`src/types/wails.d.ts`)

Глобальные декларации для Wails runtime API с полными типами для:
- `window.runtime` - Wails runtime API
- `window.go.main.App` - Go backend bindings

## Кастомные хуки

### `useWails.ts`
- `useWailsRuntime()` - доступ к Wails runtime
- `useBackend()` - доступ к backend bindings
- `useBackendCall()` - типизированные вызовы backend методов
- `useWailsEvent()` - подписка на Wails события
- `useWindow()` - управление окном приложения

### `AppContext.tsx`
React Context для глобального состояния приложения с типами:
- Управление темой
- Конфигурация приложения
- Состояние загрузки

## TypeScript конфигурация

### `tsconfig.json`
- Строгие настройки типизации
- Path mapping для удобного импорта
- Поддержка React JSX
- Оптимизации для bundler mode

### Скрипты
```json
{
  "type-check": "tsc --noEmit",     // Проверка типов без компиляции
  "build": "tsc && vite build",     // Полная сборка с проверкой типов
  "lint": "eslint . --ext ts,tsx"   // Линтинг TypeScript файлов
}
```

## Лучшие практики

### 1. Строгая типизация
```typescript
// ✅ Хорошо
interface User {
  id: number
  name: string
  email: string
}

function createUser(userData: User): Promise<User> {
  // ...
}

// ❌ Плохо
function createUser(userData: any): Promise<any> {
  // ...
}
```

### 2. Использование утилитарных типов
```typescript
// Частичные обновления
type PartialUser = Partial<User>

// Обязательные поля
type RequiredUser = Required<User>

// Только определенные поля
type UserBasic = Pick<User, 'id' | 'name'>
```

### 3. Дженерики для переиспользовемых компонентов
```typescript
interface SelectProps<T> {
  options: T[]
  value: T | null
  onChange: (value: T) => void
  renderOption: (option: T) => React.ReactNode
}

function Select<T>({ options, value, onChange, renderOption }: SelectProps<T>) {
  // ...
}
```

## Преимущества TypeScript

### 🛡️ Типобезопасность
- Ловля ошибок на этапе компиляции
- Автодополнение в IDE
- Рефакторинг с уверенностью

### 📚 Самодокументируемый код
- Интерфейсы как документация
- Явные контракты между компонентами
- Лучшее понимание API

### 🐛 Меньше багов
- Строгая типизация предотвращает распространенные ошибки
- Лучшая поддержка IDE
- Упрощенное тестирование

### 🚀 Производительность разработки
- Быстрое выявление ошибок
- Улучшенное автодополнение
- Лучший рефакторинг

## Интеграция с Wails

TypeScript обеспечивает полную типобезопасность при взаимодействии с Go backend:

```typescript
// Типизированный вызов backend метода
const { data, loading, error, execute } = useBackendCall(
  backend.getVersion // типизированная функция
)

// Типизированные события
useWailsEvent<string>('version-updated', (newVersion) => {
  console.log('New version:', newVersion) // newVersion: string
})
```

## Проверка типов

Запустите проверку типов перед коммитом:
```bash
npm run type-check
```

Это обеспечит, что весь код соответствует строгим TypeScript правилам и готов к production.
