# Установка WINE для сборки QMLauncher для Windows

Для создания NSIS установщика на Linux требуется WINE.

## Быстрая установка

```bash
sudo apt-get update
sudo apt-get install -y wine64 wine32 winetricks
```

## Первоначальная настройка WINE (один раз)

```bash
winecfg
# В открывшемся окне выберите Windows 10 и закройте
```

## Сборка установщика

После установки WINE:

```bash
cd QMLauncher
npm run dist:win
```

Установщик будет создан в папке `release/` с именем `QMLauncher Setup 1.0.0.exe`

## Альтернатива: Использовать уже собранную версию

Если не хотите устанавливать WINE, можно использовать уже собранную папку:
- `release/win-unpacked/` - содержит все необходимые файлы
- Просто скопируйте папку на Windows и запустите `QMLauncher.exe`

