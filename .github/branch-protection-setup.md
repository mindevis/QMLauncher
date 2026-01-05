# Настройка защиты веток (Branch Protection Rules)

## Автоматическая настройка

### Используя GitHub CLI

```bash
# Настройка защиты для ветки main
gh api repos/:owner/:repo/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["CI"]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1}' \
  --field restrictions=null \
  --field allow_force_pushes=false \
  --field allow_deletions=false \
  --field block_creations=false
```

### Ручная настройка через веб-интерфейс

1. Перейти в **Settings** → **Branches** → **Add rule**
2. **Branch name pattern**: `main`
3. Включить опции:
   - ✅ **Require pull request reviews before merging**
   - ✅ **Require status checks to pass before merging**
     - Status checks: `CI`
   - ✅ **Require branches to be up to date before merging**
   - ✅ **Include administrators**
   - ✅ **Restrict pushes that create matching branches**

## Проверка настроек

```bash
# Проверить текущие правила защиты
gh api repos/:owner/:repo/branches/main/protection
```

## Важные замечания

- **Только мерджи из `dev`** разрешены в `main`
- **CI должен проходить** перед мержем
- **Code review обязателен** для всех PR в `main`
- **Администраторы тоже подчиняются** правилам (enforce_admins=true)
