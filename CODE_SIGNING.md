# Подпись Windows .exe файлов

## Использование SignPath.io (настроено)

Проект настроен для использования **SignPath.io** - бесплатного сервиса для подписи кода open source проектов.

### Настройка SignPath.io

1. **Регистрация на SignPath.io**
   - Перейдите на https://signpath.io/
   - Зарегистрируйтесь (бесплатно для open source)
   - Создайте Organization
   - Создайте Project

2. **Получение API ключей**
   - В настройках проекта скопируйте:
     - `Organization ID`
     - `Project ID`
     - `API Token`

3. **Добавление секретов в GitHub**
   - Перейдите в Settings → Secrets and variables → Actions
   - Добавьте секреты:
     - `SIGNPATH_ORGANIZATION_ID` - ID организации
     - `SIGNPATH_PROJECT_ID` - ID проекта
     - `SIGNPATH_API_TOKEN` - API токен

4. **Автоматическая подпись**
   - После добавления секретов, GitHub Actions автоматически подпишет .exe файлы
   - Подпись выполняется после сборки установщика

### Альтернативные варианты

#### Коммерческие сертификаты
- **DigiCert** - https://www.digicert.com/
- **Sectigo (Comodo)** - https://sectigo.com/
- **GlobalSign** - https://www.globalsign.com/
- **Стоимость**: ~$200-400/год

#### Self-signed сертификаты (только для тестирования)
- Не рекомендуется для распространения
- Браузеры и Windows будут показывать предупреждения

## Текущая конфигурация (SignPath.io)

### package.json

```json
{
  "build": {
    "win": {
      "sign": "${env.SIGNPATH_PROJECT_ID ? 'signpath' : false}",
      "signingHashAlgorithms": ["sha256"],
      "signDlls": true,
      "signAndEditExecutable": true
    }
  }
}
```

### GitHub Actions

Workflow автоматически:
1. Проверяет наличие секретов SignPath.io
2. Устанавливает SignPath CLI (если секреты присутствуют)
3. Подписывает .exe файлы через SignPath.io API

### Переменные окружения

Workflow использует следующие секреты:
- `SIGNPATH_ORGANIZATION_ID` - ID организации в SignPath.io
- `SIGNPATH_PROJECT_ID` - ID проекта в SignPath.io
- `SIGNPATH_API_TOKEN` - API токен для аутентификации

## Настройка для коммерческих сертификатов

Если нужно использовать собственный сертификат вместо SignPath.io:

### 1. Установка сертификата

```bash
# Импорт сертификата из .pfx файла
certutil -importPFX certificate.pfx
```

### 2. Настройка package.json

```json
{
  "build": {
    "win": {
      "sign": true,
      "signingHashAlgorithms": ["sha256"],
      "signDlls": true,
      "certificateFile": "${env.CERTIFICATE_PFX}",
      "certificatePassword": "${env.CERTIFICATE_PASSWORD}"
    }
  }
}
```

### 3. Настройка GitHub Actions

```yaml
- name: Import certificate
  run: |
    $cert = [Convert]::ToBase64String([IO.File]::ReadAllBytes("${{ secrets.CERTIFICATE_PFX }}"))
    [IO.File]::WriteAllBytes("certificate.pfx", [Convert]::FromBase64String($cert))
    $password = "${{ secrets.CERTIFICATE_PASSWORD }}"
    Import-PfxCertificate -FilePath certificate.pfx -CertStoreLocation Cert:\CurrentUser\My -Password (ConvertTo-SecureString -String $password -Force -AsPlainText)

- name: Build Windows installer
  env:
    CSC_LINK: certificate.pfx
    CSC_KEY_PASSWORD: ${{ secrets.CERTIFICATE_PASSWORD }}
  run: npm run dist:win
```

## Настройка секретов в GitHub

1. Перейдите в Settings → Secrets and variables → Actions
2. Добавьте секреты:
   - `CERTIFICATE_PFX` - содержимое .pfx файла (base64)
   - `CERTIFICATE_PASSWORD` - пароль от сертификата

## Конвертация сертификата в base64

```bash
# Linux/Mac
base64 -i certificate.pfx -o certificate_base64.txt

# Windows PowerShell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("certificate.pfx"))
```

## Текущая конфигурация

В текущей конфигурации подпись отключена:

```json
"win": {
  "sign": false,
  "signingHashAlgorithms": null,
  "signDlls": false
}
```

Для включения подписи нужно:
1. Получить сертификат
2. Добавить секреты в GitHub
3. Обновить конфигурацию в package.json
4. Обновить GitHub Actions workflow

## Альтернатива: Автоматическая подпись через сервисы

- **SignPath.io** - автоматическая подпись для CI/CD
- **Azure Key Vault** - хранение и использование сертификатов
- **AWS Certificate Manager** - управление сертификатами

