# QMLauncher Frontend - React + TypeScript + Vite

Modern desktop application frontend built with React, TypeScript, Vite, and shadcn/ui components.

## 🚀 Tech Stack

- **React 18** - Modern React with hooks and concurrent features
- **TypeScript** - Full type safety and excellent developer experience
- **Vite** - Fast build tool and development server
- **shadcn/ui** - Beautiful and accessible UI components
- **Tailwind CSS** - Utility-first CSS framework
- **Wails** - Desktop application framework integration

## 🛠️ Development

### Prerequisites

- Node.js 18+
- npm or yarn

### Installation

```bash
npm install
```

### Development Server

```bash
npm run dev
```

### Build for Production

```bash
npm run build
```

### Type Checking

```bash
npm run type-check
```

### Linting

```bash
npm run lint
```

## 📁 Project Structure

```
src/
├── app/                    # Next.js app directory (if using)
├── components/
│   ├── ui/                # shadcn/ui components
│   └── ...                # Custom components
├── contexts/              # React contexts
├── hooks/                 # Custom hooks
├── lib/                   # Utilities and configs
├── types/                 # TypeScript type definitions
├── App.tsx               # Main app component
├── main.tsx              # React entry point
└── index.css             # Global styles
```

## 🎨 UI Components

This project uses [shadcn/ui](https://ui.shadcn.com/) for consistent and beautiful UI components.

### Adding New Components

```bash
npx shadcn@latest add [component-name]
```

Available components: button, card, dialog, dropdown-menu, etc.

## 🔧 TypeScript

Full TypeScript support with strict type checking. See [TYPESCRIPT_README.md](./TYPESCRIPT_README.md) for detailed information about type definitions and best practices.

### Key Features

- Strict type checking
- Custom hooks with full typing
- Wails backend integration types
- Context API with TypeScript
- Path mapping for clean imports

## 🎯 Wails Integration

The frontend is designed to work seamlessly with the Wails desktop application framework.

### Backend Communication

```typescript
import { useBackend, useWailsEvent } from '@/hooks/useWails'

// Access backend methods
const backend = useBackend()

// Listen to backend events
useWailsEvent('app-ready', (data) => {
  console.log('App is ready:', data)
})
```

## 📝 Scripts

- `dev` - Start development server
- `build` - Build for production
- `preview` - Preview production build
- `type-check` - Run TypeScript type checking
- `lint` - Run ESLint

## 🔍 IDE Setup

### VS Code (Recommended)

- [TypeScript Importer](https://marketplace.visualstudio.com/items?itemName=pmneo.tsimporter)
- [Tailwind CSS IntelliSense](https://marketplace.visualstudio.com/items?itemName=bradlc.vscode-tailwindcss)
- [ESLint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint)

### IntelliSense

Full IntelliSense support for:
- React components and props
- TypeScript types and interfaces
- Tailwind CSS classes
- Wails backend methods

## 🚀 Deployment

The built application is automatically bundled by Wails for desktop distribution.

```bash
# In the root project directory
wails build
```
