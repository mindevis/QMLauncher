# 🔀 Git Workflow & Branching Guide

> Подробное руководство по работе с ветками в проекте QMLauncher

## 📖 Быстрый старт

Если вы новичок в Git Flow, начните с [базовых команд](#-ежедневные-команды-git-flow) ниже.

---

## 🎯 Цели Git Flow в проекте

- **Стабильность**: `main` всегда содержит рабочий код
- **Параллельная разработка**: несколько фич одновременно
- **Предсказуемые релизы**: контролируемый процесс релизов
- **Безопасность**: защита от случайных изменений в production

---

## 📋 Ежедневные команды Git Flow

### 🔍 Информация о ветках
```bash
# Текущая ветка и статус
git status

# Все ветки
git branch -a

# Граф коммитов
git log --oneline --graph --all -5
```

### 🔄 Переключение веток
```bash
# На существующую ветку
git checkout dev
git checkout feature/my-work

# Создать и переключиться
git checkout -b feature/new-feature

# Вернуться назад
git checkout -
```

### 🌟 Feature ветки
```bash
# Начать работу
git flow feature start user-login

# Закончить (автоматически сольет в dev)
git flow feature finish user-login

# Для командной работы
git flow feature publish user-login
git flow feature pull origin user-login
```

---

## 🚀 Полный рабочий процесс

### 1. Настройка (один раз)
```bash
# Клонировать
git clone https://github.com/mindevis/QMLauncher.git
cd QMLauncher

# Настроить upstream
git remote add upstream https://github.com/mindevis/QMLauncher.git
```

### 2. Начало работы
```bash
# Обновить dev
git checkout dev
git pull origin dev

# Начать feature
git flow feature start add-dark-theme
```

### 3. Разработка
```bash
# Работа...
git add .
git commit -m "feat: add dark theme toggle"

# Push для review
git push origin feature/add-dark-theme
```

### 4. Code Review
```bash
# Создать PR в dev ветку
# После approval:
git flow feature finish add-dark-theme
```

---

## 🛠️ Продвинутые сценарии

### Срочное исправление (Hotfix)
```bash
# Из production
git flow hotfix start fix-crash
# Исправить баг
git flow hotfix finish fix-crash
```

### Работа с несколькими фичами
```bash
# Переключаться между задачами
git checkout feature/task-1
# Работа...
git checkout feature/task-2
# Работа...
```

### Синхронизация с командой
```bash
# Обновить все ветки
git fetch origin

# Обновить текущую ветку
git pull origin dev

# Push своих изменений
git push origin feature/my-work
```

---

## ⚠️ Частые проблемы и решения

### "Branch already exists"
```bash
git branch -D feature/existing-branch
git flow feature start new-name
```

### Забытые изменения
```bash
# Найти последний коммит
git log --oneline -5

# Создать новую ветку оттуда
git checkout -b feature/correct-branch <commit-hash>
```

### Конфликты при merge
```bash
# Разрешить конфликты в файлах
git add <resolved-files>
git commit
```

---

## 📊 Визуальная схема

```
main (production) ──┬── hotfix/crash-fix ──┐
                   │                      ├── tag v1.1.1
dev (development) ─┼── release/v1.1.0 ────┘
                   │
                   ├── feature/user-auth ──┐
                   ├── feature/dark-theme ─┼── merge to dev
                   └── feature/api-v2 ─────┘
```

---

## 🔗 Ссылки

- [Git Flow Documentation](https://nvie.com/posts/a-successful-git-branching-model/)
- [Conventional Commits](https://conventionalcommits.org/)
- [CONTRIBUTING.md](../CONTRIBUTING.md) - основные правила проекта

---

*Обновлено: $(date)*