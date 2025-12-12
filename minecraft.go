package main

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	CONCURRENCY        = 8
	ASSET_CONCURRENCY  = 8
	MOJANG_VERSION_URL = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
)

type MinecraftService struct {
	ctx        context.Context
	app        *App
	process    *exec.Cmd
	processMux sync.Mutex
}

func NewMinecraftService(app *App) *MinecraftService {
	return &MinecraftService{
		app: app,
	}
}

func (m *MinecraftService) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// Startup вызывается при запуске приложения
func (m *MinecraftService) Startup(ctx context.Context) {
	m.ctx = ctx
}

// LaunchMinecraft запускает Minecraft
func (m *MinecraftService) LaunchMinecraft(args LaunchMinecraftArgs) (*LaunchResult, error) {
	log.Printf("[Minecraft] Начало запуска Minecraft: version=%s, serverUuid=%s, username=%s", args.MinecraftVersion, args.ServerUuid, args.Username)
	log.Printf("[Minecraft] LaunchMinecraftArgs: JavaPath=%s, WorkingDirectory=%s, HWID=%s, GameArgs count=%d, JVMArgs count=%d",
		args.JavaPath, args.WorkingDirectory, args.HWID, len(args.GameArgs), len(args.JVMArgs))

	m.processMux.Lock()
	defer m.processMux.Unlock()

	if m.process != nil {
		log.Printf("[Minecraft] Ошибка: Minecraft уже запущен")
		return nil, fmt.Errorf("minecraft уже запущен")
	}

	// Получаем настройки
	settings, err := m.app.GetSettings()
	if err != nil {
		log.Printf("[Minecraft] Ошибка получения настроек: %v", err)
		return nil, fmt.Errorf("ошибка получения настроек: %v", err)
	}

	// Определяем путь к Minecraft
	minecraftBasePath := m.getMinecraftBasePath(settings, args.ServerUuid)
	minecraftBasePath = expandPath(minecraftBasePath)

	// Рабочая директория: используем то, что пришло, только если существует; иначе fallback на корень minecraft
	workingDir := expandPath(args.WorkingDirectory)
	if workingDir == "" || !dirExists(workingDir) {
		if workingDir != "" && !dirExists(workingDir) {
			log.Printf("[Minecraft] Рабочая директория из фронта не найдена: %s, используем базовую", workingDir)
		}
		workingDir = minecraftBasePath
	}
	log.Printf("[Minecraft] Путь к Minecraft: %s", minecraftBasePath)
	log.Printf("[Minecraft] Рабочая директория: %s", workingDir)

	// Получаем путь к Java
	log.Printf("[Minecraft] Получение пути к Java для serverUuid=%s...", args.ServerUuid)
	javaPath, err := m.app.GetJavaPath(args.ServerUuid)
	if err != nil {
		log.Printf("[Minecraft] Ошибка получения пути к Java: %v", err)
		return nil, fmt.Errorf("ошибка получения пути к Java: %v", err)
	}
	javaPath = expandPath(javaPath)

	// На Windows используем javaw.exe вместо java.exe, чтобы не показывать консоль
	if runtime.GOOS == "windows" && strings.HasSuffix(javaPath, "java.exe") {
		javawPath := strings.TrimSuffix(javaPath, "java.exe") + "javaw.exe"
		// Проверяем, существует ли javaw.exe, если нет - используем java.exe
		if _, err := os.Stat(javawPath); err == nil {
			javaPath = javawPath
			log.Printf("[Minecraft] Используется javaw.exe для скрытия консоли: %s", javaPath)
		}
	}

	log.Printf("[Minecraft] Путь к Java: %s", javaPath)

	// Проверяем права на выполнение
	if err := os.Chmod(javaPath, 0755); err != nil {
		log.Printf("[Minecraft] Предупреждение: не удалось установить права на выполнение для Java: %v", err)
		// Не критично, продолжаем
	}

	// Читаем version.json
	versionDir := filepath.Join(minecraftBasePath, "versions", args.MinecraftVersion)
	versionJsonPath := filepath.Join(versionDir, args.MinecraftVersion+".json")
	log.Printf("[Minecraft] Загрузка version.json из %s...", versionJsonPath)

	versionData, err := m.loadVersionJson(versionJsonPath)
	if err != nil {
		log.Printf("[Minecraft] Ошибка загрузки version.json: %v", err)
		return nil, fmt.Errorf("ошибка загрузки version.json: %v", err)
	}
	log.Printf("[Minecraft] version.json загружен: MainClass=%s, Libraries=%d", versionData.MainClass, len(versionData.Libraries))

	// Кэшируем classpath один раз, чтобы не вызывать buildClasspath многократно
	classpathCache := m.buildClasspath(versionData, minecraftBasePath)

	// Строим JVM аргументы
	log.Printf("[Minecraft] Построение JVM аргументов...")
	jvmArgs := m.buildJVMArgs(versionData, settings, args, minecraftBasePath, classpathCache)
	log.Printf("[Minecraft] JVM аргументов: %d", len(jvmArgs))
	for i, arg := range jvmArgs {
		if i < 5 || i >= len(jvmArgs)-5 {
			log.Printf("[Minecraft]   JVM[%d]: %s", i, arg)
		} else if i == 5 {
			log.Printf("[Minecraft]   ... (пропущено %d аргументов) ...", len(jvmArgs)-10)
		}
	}

	// Строим игровые аргументы
	log.Printf("[Minecraft] Построение игровых аргументов...")
	gameArgsFromVersion := m.buildGameArgs(versionData, args, minecraftBasePath, classpathCache)

	// Если есть аргументы с фронтенда, используем их полностью (они уже содержат все необходимое)
	// Иначе используем аргументы из version.json
	var gameArgs []string
	if len(args.GameArgs) > 0 {
		log.Printf("[Minecraft] Использование %d аргументов с фронтенда (полная замена аргументов из version.json)", len(args.GameArgs))
		gameArgs = args.GameArgs
	} else {
		log.Printf("[Minecraft] Использование аргументов из version.json")
		gameArgs = gameArgsFromVersion
	}

	// Нормализуем пути и принудительно выставляем корректные gameDir / assetsDir / assetIndex
	assetsDir := filepath.Join(minecraftBasePath, "assets")
	assetIndexID := versionData.AssetIndex.ID

	// Проверяем наличие --server и --port в аргументах
	hasServer := false
	hasPort := false
	for i := 0; i < len(gameArgs); i++ {
		switch gameArgs[i] {
		case "--gameDir":
			if i+1 < len(gameArgs) {
				gameArgs[i+1] = expandPath(minecraftBasePath)
			}
		case "--assetsDir":
			if i+1 < len(gameArgs) {
				gameArgs[i+1] = expandPath(assetsDir)
			}
		case "--assetIndex":
			if i+1 < len(gameArgs) {
				gameArgs[i+1] = assetIndexID
			}
		case "--server":
			hasServer = true
			if i+1 < len(gameArgs) {
				// Разворачиваем путь, если нужно
				if strings.HasPrefix(gameArgs[i+1], "~") {
					gameArgs[i+1] = expandPath(gameArgs[i+1])
				}
				log.Printf("[Minecraft] Найден аргумент --server: %s", gameArgs[i+1])
			}
		case "--port":
			hasPort = true
			if i+1 < len(gameArgs) {
				log.Printf("[Minecraft] Найден аргумент --port: %s", gameArgs[i+1])
			}
		case "--quickPlayMultiplayer":
			hasServer = true // quickPlayMultiplayer также означает подключение к серверу
			hasPort = true
			if i+1 < len(gameArgs) {
				log.Printf("[Minecraft] Найден аргумент --quickPlayMultiplayer: %s", gameArgs[i+1])
				// Формат: server:port (например, "example.com:25565")
				if strings.Contains(gameArgs[i+1], ":") {
					parts := strings.Split(gameArgs[i+1], ":")
					if len(parts) == 2 {
						log.Printf("[Minecraft] quickPlayMultiplayer: server=%s, port=%s", parts[0], parts[1])
					}
				}
			}
		default:
			// Если аргумент сам по себе путь с тильдой, разворачиваем
			if strings.HasPrefix(gameArgs[i], "~") {
				gameArgs[i] = expandPath(gameArgs[i])
			}
		}
	}

	// Логируем наличие аргументов сервера
	if hasServer && hasPort {
		log.Printf("[Minecraft] Аргументы --server и --port присутствуют, автоматическое подключение будет выполнено")
	} else {
		if !hasServer {
			log.Printf("[Minecraft] Предупреждение: аргумент --server отсутствует, автоматическое подключение не будет выполнено")
		}
		if !hasPort {
			log.Printf("[Minecraft] Предупреждение: аргумент --port отсутствует, автоматическое подключение не будет выполнено")
		}
	}

	log.Printf("[Minecraft] Игровых аргументов: %d", len(gameArgs))
	for i, arg := range gameArgs {
		if i < 5 || i >= len(gameArgs)-5 {
			log.Printf("[Minecraft]   Game[%d]: %s", i, arg)
		} else if i == 5 {
			log.Printf("[Minecraft]   ... (пропущено %d аргументов) ...", len(gameArgs)-10)
		}
	}

	// Запускаем процесс
	allArgs := append(jvmArgs, versionData.MainClass)
	allArgs = append(allArgs, gameArgs...)
	log.Printf("[Minecraft] Всего аргументов: %d (JVM: %d + MainClass + Game: %d)", len(allArgs), len(jvmArgs), len(gameArgs))

	log.Printf("[Minecraft] Запуск процесса: %s", javaPath)
	m.process = exec.Command(javaPath, allArgs...)
	m.process.Dir = workingDir

	// Скрываем консоль на Windows ДО создания pipe'ов
	// Это устанавливает CREATE_NO_WINDOW флаг, чтобы Java процесс не показывал консоль
	setCmdHideWindow(m.process)

	// Создаем буферы для вывода (pipe'ы не показывают консоль, в отличие от os.Stdout/os.Stderr)
	stdoutPipe, err := m.process.StdoutPipe()
	if err != nil {
		log.Printf("[Minecraft] Ошибка создания stdout pipe: %v", err)
		// На Windows не перенаправляем в os.Stdout, чтобы не показывать консоль
		// Вместо этого используем io.Discard для подавления вывода
		if runtime.GOOS == "windows" {
			m.process.Stdout = nil // nil означает, что вывод будет подавлен
		} else {
			m.process.Stdout = os.Stdout
		}
	} else {
		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			lineCount := 0
			for scanner.Scan() {
				line := scanner.Text()
				lineCount++
				if lineCount <= 50 { // Логируем первые 50 строк
					log.Printf("[Minecraft] STDOUT: %s", line)
				} else if lineCount == 51 {
					log.Printf("[Minecraft] STDOUT: ... (дальше вывод идет только в лог)")
				}
				// На Windows не выводим в консоль, чтобы не показывать окно
				if runtime.GOOS != "windows" {
					fmt.Println(line) // Выводим в консоль только на не-Windows платформах
				}
			}
		}()
	}

	stderrPipe, err := m.process.StderrPipe()
	if err != nil {
		log.Printf("[Minecraft] Ошибка создания stderr pipe: %v", err)
		// На Windows не перенаправляем в os.Stderr, чтобы не показывать консоль
		// Вместо этого используем nil для подавления вывода
		if runtime.GOOS == "windows" {
			m.process.Stderr = nil // nil означает, что вывод будет подавлен
		} else {
			m.process.Stderr = os.Stderr
		}
	} else {
		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			lineCount := 0
			for scanner.Scan() {
				line := scanner.Text()
				lineCount++
				if lineCount <= 50 { // Логируем первые 50 строк
					log.Printf("[Minecraft] STDERR: %s", line)
				} else if lineCount == 51 {
					log.Printf("[Minecraft] STDERR: ... (дальше вывод идет только в лог)")
				}
				// На Windows не выводим в консоль, чтобы не показывать окно
				if runtime.GOOS != "windows" {
					fmt.Fprintln(os.Stderr, line) // Выводим в stderr только на не-Windows платформах
				}
			}
		}()
	}

	if err := m.process.Start(); err != nil {
		m.process = nil
		log.Printf("[Minecraft] Ошибка запуска процесса: %v", err)
		return nil, fmt.Errorf("ошибка запуска Minecraft: %v", err)
	}

	log.Printf("[Minecraft] Процесс запущен успешно, PID: %d", m.process.Process.Pid)

	// Отслеживаем завершение процесса
	go func() {
		log.Printf("[Minecraft] Ожидание завершения процесса (PID: %d)...", m.process.Process.Pid)
		err := m.process.Wait()
		m.processMux.Lock()
		if err != nil {
			log.Printf("[Minecraft] Процесс завершился с ошибкой: %v", err)
		} else {
			log.Printf("[Minecraft] Процесс завершился успешно")
		}
		m.process = nil
		m.processMux.Unlock()
		log.Printf("[Minecraft] Процесс очищен из памяти")
	}()

	return &LaunchResult{Success: true}, nil
}

// IsMinecraftRunning проверяет, запущен ли процесс Minecraft
func (m *MinecraftService) IsMinecraftRunning() bool {
	m.processMux.Lock()
	defer m.processMux.Unlock()

	// Если процесс nil, значит он не запущен или уже завершился
	if m.process == nil || m.process.Process == nil {
		return false
	}

	// Проверяем, что процесс еще существует
	// На Windows и Unix используем os.FindProcess для проверки
	pid := m.process.Process.Pid
	process, err := os.FindProcess(pid)
	if err != nil {
		// Процесс не найден
		m.process = nil
		return false
	}

	// Проверяем состояние процесса
	// На Unix-системах можно использовать process.Signal(syscall.Signal(0))
	// но это может не работать на всех системах
	// Проще всего - проверить, что процесс не nil и полагаться на горутину Wait(),
	// которая установит m.process = nil при завершении
	// Для более точной проверки можно использовать runtime.GOOS-специфичные методы,
	// но для нашей задачи достаточно проверки, что процесс не nil
	_ = process // Используем переменную, чтобы избежать ошибки компиляции
	return true
}

// StopMinecraft останавливает Minecraft
func (m *MinecraftService) StopMinecraft() error {
	log.Printf("[Minecraft] Запрос на остановку Minecraft...")
	m.processMux.Lock()
	defer m.processMux.Unlock()

	if m.process == nil {
		log.Printf("[Minecraft] Minecraft не запущен, остановка не требуется")
		return fmt.Errorf("minecraft не запущен")
	}

	pid := m.process.Process.Pid
	log.Printf("[Minecraft] Остановка процесса (PID: %d)...", pid)
	if err := m.process.Process.Kill(); err != nil {
		log.Printf("[Minecraft] Ошибка остановки процесса: %v", err)
		return fmt.Errorf("ошибка остановки процесса: %v", err)
	}

	m.process = nil
	log.Printf("[Minecraft] Процесс остановлен и очищен из памяти")
	return nil
}

// expandPath разворачивает тильду в путь домашнего каталога и нормализует разделители
func expandPath(p string) string {
	if p == "" {
		return p
	}
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(p, "~") {
		p = filepath.Join(home, strings.TrimLeft(p[1:], "/\\"))
	}
	return filepath.Clean(p)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// InstallMinecraftClient устанавливает клиент Minecraft
func (m *MinecraftService) InstallMinecraftClient(version string, javaVendor string, javaVersion string, serverUuid string) (*InstallResult, error) {
	log.Printf("[Minecraft] Начало установки клиента: version=%s, javaVendor=%s, javaVersion=%s, serverUuid=%s", version, javaVendor, javaVersion, serverUuid)

	settings, err := m.app.GetSettings()
	if err != nil {
		log.Printf("[Minecraft] Ошибка получения настроек: %v", err)
		return nil, fmt.Errorf("ошибка получения настроек: %v", err)
	}

	serverUuidForPath := serverUuid
	if serverUuidForPath == "" {
		serverUuidForPath = "global"
	}
	log.Printf("[Minecraft] Используется serverUuid для пути: %s", serverUuidForPath)

	// Устанавливаем Java сначала
	log.Printf("[Minecraft] Установка Java: vendor=%s, version=%s", javaVendor, javaVersion)
	if err := m.app.InstallJava(javaVendor, javaVersion, serverUuidForPath); err != nil {
		log.Printf("[Minecraft] Ошибка установки Java: %v", err)
		return nil, fmt.Errorf("ошибка установки Java: %v", err)
	}
	log.Printf("[Minecraft] Java установлена успешно")

	// Определяем путь к Minecraft
	minecraftBasePath := m.getMinecraftBasePath(settings, serverUuidForPath)
	versionsDir := filepath.Join(minecraftBasePath, "versions")
	versionDir := filepath.Join(versionsDir, version)
	clientJar := filepath.Join(versionDir, version+".jar")
	librariesDir := filepath.Join(minecraftBasePath, "libraries")
	assetsDir := filepath.Join(minecraftBasePath, "assets")

	log.Printf("[Minecraft] Пути установки:")
	log.Printf("[Minecraft]   Base: %s", minecraftBasePath)
	log.Printf("[Minecraft]   Version dir: %s", versionDir)
	log.Printf("[Minecraft]   Client JAR: %s", clientJar)
	log.Printf("[Minecraft]   Libraries: %s", librariesDir)
	log.Printf("[Minecraft]   Assets: %s", assetsDir)

	// Создаем директории
	log.Printf("[Minecraft] Создание директорий...")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		log.Printf("[Minecraft] Ошибка создания директории версии: %v", err)
		return nil, fmt.Errorf("ошибка создания директории версии: %v", err)
	}
	if err := os.MkdirAll(librariesDir, 0755); err != nil {
		log.Printf("[Minecraft] Ошибка создания директории библиотек: %v", err)
		return nil, fmt.Errorf("ошибка создания директории библиотек: %v", err)
	}
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		log.Printf("[Minecraft] Ошибка создания директории ресурсов: %v", err)
		return nil, fmt.Errorf("ошибка создания директории ресурсов: %v", err)
	}
	log.Printf("[Minecraft] Директории созданы")

	// Проверяем, не установлен ли уже
	if _, err := os.Stat(clientJar); err == nil {
		log.Printf("[Minecraft] Клиент уже установлен: %s", clientJar)
		return &InstallResult{
			Success:          true,
			AlreadyInstalled: true,
			Message:          fmt.Sprintf("Minecraft %s уже установлен", version),
		}, nil
	}

	// Получаем манифест версий
	log.Printf("[Minecraft] Получение манифеста версий...")
	manifest, err := m.fetchVersionManifest()
	if err != nil {
		log.Printf("[Minecraft] Ошибка получения манифеста: %v", err)
		return nil, fmt.Errorf("ошибка получения манифеста: %v", err)
	}
	log.Printf("[Minecraft] Манифест получен, версий в манифесте: %d", len(manifest.Versions))

	// Находим нужную версию
	var versionInfo *VersionInfo
	for _, v := range manifest.Versions {
		if v.ID == version {
			versionInfo = &v
			break
		}
	}
	if versionInfo == nil {
		log.Printf("[Minecraft] Версия %s не найдена в манифесте", version)
		return nil, fmt.Errorf("версия %s не найдена", version)
	}
	log.Printf("[Minecraft] Версия найдена в манифесте: URL=%s", versionInfo.URL)

	// Загружаем информацию о версии
	log.Printf("[Minecraft] Загрузка данных версии из %s...", versionInfo.URL)
	versionData, err := m.fetchVersionData(versionInfo.URL)
	if err != nil {
		log.Printf("[Minecraft] Ошибка загрузки данных версии: %v", err)
		return nil, fmt.Errorf("ошибка загрузки данных версии: %v", err)
	}
	log.Printf("[Minecraft] Данные версии загружены: MainClass=%s, Libraries=%d", versionData.MainClass, len(versionData.Libraries))

	// Сохраняем version.json
	versionJsonPath := filepath.Join(versionDir, version+".json")
	log.Printf("[Minecraft] Сохранение version.json в %s...", versionJsonPath)
	versionJsonData, _ := json.MarshalIndent(versionData, "", "  ")
	if err := os.WriteFile(versionJsonPath, versionJsonData, 0644); err != nil {
		log.Printf("[Minecraft] Ошибка сохранения version.json: %v", err)
		return nil, fmt.Errorf("ошибка сохранения version.json: %v", err)
	}
	log.Printf("[Minecraft] version.json сохранен")

	// Загружаем клиент JAR
	log.Printf("[Minecraft] Загрузка клиентского JAR из %s в %s...", versionData.Downloads.Client.URL, clientJar)
	if err := m.downloadFile(versionData.Downloads.Client.URL, clientJar, versionData.Downloads.Client.SHA1); err != nil {
		log.Printf("[Minecraft] Ошибка загрузки клиента: %v", err)
		return nil, fmt.Errorf("ошибка загрузки клиента: %v", err)
	}
	log.Printf("[Minecraft] Клиентский JAR загружен: %s", clientJar)

	// Загружаем библиотеки параллельно
	log.Printf("[Minecraft] Загрузка библиотек (всего: %d)...", len(versionData.Libraries))
	if err := m.downloadLibraries(versionData, librariesDir); err != nil {
		log.Printf("[Minecraft] Ошибка загрузки библиотек: %v", err)
		return nil, fmt.Errorf("ошибка загрузки библиотек: %v", err)
	}
	log.Printf("[Minecraft] Библиотеки загружены")

	// Гарантируем создание директории natives и дополнительный проход по извлечению natives
	nativesDir := filepath.Join(librariesDir, "natives")
	if err := os.MkdirAll(nativesDir, 0755); err != nil {
		log.Printf("[Minecraft] ПРЕДУПРЕЖДЕНИЕ: не удалось создать директорию natives: %v", err)
	} else {
		extracted := m.extractAllNatives(versionData, librariesDir)
		log.Printf("[Minecraft] Проверка natives после загрузки: директория=%s, повторных извлечений=%d", nativesDir, extracted)
	}

	// Загружаем assets
	log.Printf("[Minecraft] Загрузка ресурсов (assetIndex: %s)...", versionData.AssetIndex.ID)
	if err := m.downloadAssets(versionData, assetsDir); err != nil {
		log.Printf("[Minecraft] Ошибка загрузки ресурсов: %v", err)
		return nil, fmt.Errorf("ошибка загрузки ресурсов: %v", err)
	}
	log.Printf("[Minecraft] Ресурсы загружены")

	// Проверяем, что natives извлечены
	if entries, err := os.ReadDir(nativesDir); err == nil {
		dllCount := 0
		soCount := 0
		dylibCount := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				name := entry.Name()
				if strings.HasSuffix(name, ".dll") {
					dllCount++
				} else if strings.HasSuffix(name, ".so") {
					soCount++
				} else if strings.HasSuffix(name, ".dylib") {
					dylibCount++
				}
			}
		}
		log.Printf("[Minecraft] Проверка natives: найдено файлов в %s - DLL: %d, SO: %d, DYLIB: %d", nativesDir, dllCount, soCount, dylibCount)
		if runtime.GOOS == "windows" && dllCount == 0 {
			log.Printf("[Minecraft] ПРЕДУПРЕЖДЕНИЕ: не найдено DLL файлов в natives директории для Windows! Повторная попытка извлечения...")
			reExtracted := m.extractAllNatives(versionData, librariesDir)
			log.Printf("[Minecraft] Повторная попытка извлечения завершена, извлечено: %d", reExtracted)

			// Пересчитываем
			if entries2, err2 := os.ReadDir(nativesDir); err2 == nil {
				dll2, so2, dylib2 := 0, 0, 0
				for _, e := range entries2 {
					if !e.IsDir() {
						name := e.Name()
						if strings.HasSuffix(name, ".dll") {
							dll2++
						} else if strings.HasSuffix(name, ".so") {
							so2++
						} else if strings.HasSuffix(name, ".dylib") {
							dylib2++
						}
					}
				}
				log.Printf("[Minecraft] Повторная проверка natives: DLL: %d, SO: %d, DYLIB: %d", dll2, so2, dylib2)
				if dll2 == 0 {
					return nil, fmt.Errorf("не удалось извлечь natives: отсутствуют DLL файлы")
				}
			} else {
				return nil, fmt.Errorf("не удалось перечитать natives директорию: %v", err2)
			}
		}
	} else {
		log.Printf("[Minecraft] ПРЕДУПРЕЖДЕНИЕ: не удалось прочитать natives директорию: %v", err)
	}

	log.Printf("[Minecraft] Установка клиента завершена успешно: version=%s", version)
	return &InstallResult{
		Success: true,
		Message: fmt.Sprintf("Minecraft %s успешно установлен", version),
	}, nil
}

// UninstallMinecraft удаляет установленный Minecraft клиент
func (m *MinecraftService) UninstallMinecraft(serverId int) (*UninstallResult, error) {
	log.Printf("[Minecraft] Начало удаления клиента: serverId=%d", serverId)

	// Получаем server_uuid из конфигурации
	serverUuid := ""
	log.Printf("[Minecraft] Получение server_uuid из конфигурации...")
	configService := NewConfigService(m.app)
	dbConfig, err := configService.GetLauncherDbConfig(serverId)
	if err == nil && dbConfig != nil {
		// Пробуем получить server_uuid из конфигурации
		if uuid, ok := dbConfig.Config["server_uuid"].(string); ok && uuid != "" {
			serverUuid = uuid
			log.Printf("[Minecraft] server_uuid получен из конфигурации: %s", serverUuid)
		}
	}

	// Если не получили из конфигурации, пробуем из embedded servers
	if serverUuid == "" {
		log.Printf("[Minecraft] Получение server_uuid из embedded servers...")
		embeddedServers, err := m.app.GetEmbeddedServers()
		if err == nil {
			for _, server := range embeddedServers {
				if server.ServerID == serverId {
					serverUuid = server.ServerUUID
					log.Printf("[Minecraft] server_uuid получен из embedded servers: %s", serverUuid)
					break
				}
			}
		}
	}

	// Определяем базовую директорию .qmlauncher
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
		if home == "" {
			home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		}
	}
	baseDir := filepath.Join(home, ".qmlauncher")

	// Строго ищем по server_uuid в директории .qmlauncher
	// serverId НЕ используется в иерархии папок, только server_uuid
	if serverUuid == "" {
		// Дополнительный fallback: если в .qmlauncher только один подпроект, используем его как serverUuid
		if entries, err := os.ReadDir(baseDir); err == nil {
			candidates := []string{}
			for _, e := range entries {
				if e.IsDir() {
					name := e.Name()
					// пропускаем спец-папки (logs и т.п.)
					if name == "logs" {
						continue
					}
					candidates = append(candidates, name)
				}
			}
			if len(candidates) == 1 {
				serverUuid = candidates[0]
				log.Printf("[Minecraft] server_uuid не найден в конфигурации, используем единственный найденный каталог: %s", serverUuid)
			}
		}
	}

	if serverUuid == "" {
		log.Printf("[Minecraft] Ошибка: server_uuid не определен для serverId=%d", serverId)
		log.Printf("[Minecraft] server_uuid должен быть получен из конфигурации или embedded servers")
		return &UninstallResult{
			Success: false,
			Error:   fmt.Sprintf("Не удалось определить server_uuid для serverId=%d. Убедитесь, что сервер настроен правильно.", serverId),
		}, nil
	}

	// Пути к директориям Minecraft и Java
	minecraftPath := filepath.Join(baseDir, serverUuid, "minecraft")
	javaPath := filepath.Join(baseDir, serverUuid, "java")

	log.Printf("[Minecraft] Поиск директорий для удаления:")
	log.Printf("[Minecraft]   - Minecraft: %s", minecraftPath)
	log.Printf("[Minecraft]   - Java: %s", javaPath)

	removedSomething := false

	// Удаляем директорию Minecraft
	if info, err := os.Stat(minecraftPath); err == nil && info.IsDir() {
		log.Printf("[Minecraft] Найдена директория Minecraft, начинаем удаление: %s", minecraftPath)
		if err := os.RemoveAll(minecraftPath); err != nil {
			log.Printf("[Minecraft] Ошибка удаления директории Minecraft: %v", err)
			return &UninstallResult{
				Success: false,
				Error:   fmt.Sprintf("Ошибка удаления директории Minecraft: %v", err),
			}, nil
		}
		// Проверяем, что директория удалена
		if _, err := os.Stat(minecraftPath); os.IsNotExist(err) {
			log.Printf("[Minecraft] Директория Minecraft успешно удалена: %s", minecraftPath)
			removedSomething = true
		} else {
			log.Printf("[Minecraft] Предупреждение: директория Minecraft все еще существует после удаления: %s", minecraftPath)
			return &UninstallResult{
				Success: false,
				Error:   fmt.Sprintf("Директория Minecraft все еще существует после удаления: %s", minecraftPath),
			}, nil
		}
	} else if os.IsNotExist(err) {
		log.Printf("[Minecraft] Директория Minecraft не существует: %s", minecraftPath)
	} else {
		log.Printf("[Minecraft] Ошибка проверки директории Minecraft: %v", err)
	}

	// Удаляем директорию Java
	if info, err := os.Stat(javaPath); err == nil && info.IsDir() {
		log.Printf("[Minecraft] Найдена директория Java, начинаем удаление: %s", javaPath)
		if err := os.RemoveAll(javaPath); err != nil {
			log.Printf("[Minecraft] Ошибка удаления директории Java: %v", err)
			// Не возвращаем ошибку, если Minecraft уже удален
			if !removedSomething {
				return &UninstallResult{
					Success: false,
					Error:   fmt.Sprintf("Ошибка удаления директории Java: %v", err),
				}, nil
			}
		} else {
			// Проверяем, что директория удалена
			if _, err := os.Stat(javaPath); os.IsNotExist(err) {
				log.Printf("[Minecraft] Директория Java успешно удалена: %s", javaPath)
				removedSomething = true
			} else {
				log.Printf("[Minecraft] Предупреждение: директория Java все еще существует после удаления: %s", javaPath)
			}
		}
	} else if os.IsNotExist(err) {
		log.Printf("[Minecraft] Директория Java не существует: %s", javaPath)
	} else {
		log.Printf("[Minecraft] Ошибка проверки директории Java: %v", err)
	}

	if removedSomething {
		log.Printf("[Minecraft] Удаление завершено успешно для server_uuid=%s", serverUuid)
		return &UninstallResult{Success: true}, nil
	}

	log.Printf("[Minecraft] Ничего не удалено: директории не найдены для server_uuid=%s", serverUuid)
	return &UninstallResult{
		Success: false,
		Error:   fmt.Sprintf("Директории Minecraft/Java не найдены для server_uuid=%s", serverUuid),
	}, nil
}

// CheckClientInstalled проверяет, установлен ли клиент
func (m *MinecraftService) CheckClientInstalled(serverId int, serverUuid string) (*ClientCheckResult, error) {
	settings, err := m.app.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения настроек: %v", err)
	}

	serverUuidForPath := serverUuid
	if serverUuidForPath == "" {
		serverUuidForPath = "global"
	}

	minecraftBasePath := m.getMinecraftBasePath(settings, serverUuidForPath)

	// Проверяем наличие версий
	versionsDir := filepath.Join(minecraftBasePath, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return &ClientCheckResult{
			Success:   true,
			Installed: false,
			HasClient: false,
		}, nil
	}

	hasClient := false
	for _, entry := range entries {
		if entry.IsDir() {
			versionJar := filepath.Join(versionsDir, entry.Name(), entry.Name()+".jar")
			if _, err := os.Stat(versionJar); err == nil {
				hasClient = true
				break
			}
		}
	}

	return &ClientCheckResult{
		Success:   true,
		Installed: hasClient,
		HasClient: hasClient,
	}, nil
}

// Вспомогательные методы

func (m *MinecraftService) getMinecraftBasePath(settings *Settings, serverUuid string) string {
	defaultPath := filepath.Join(os.Getenv("HOME"), ".qmlauncher", serverUuid, "minecraft")
	if runtime.GOOS == "windows" {
		home := os.Getenv("USERPROFILE")
		if home == "" {
			home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		}
		defaultPath = filepath.Join(home, ".qmlauncher", serverUuid, "minecraft")
	}

	if settings.MinecraftPath == "" {
		return defaultPath
	}

	customPath := strings.Replace(settings.MinecraftPath, "~", os.Getenv("HOME"), 1)
	if runtime.GOOS == "windows" {
		home := os.Getenv("USERPROFILE")
		customPath = strings.Replace(customPath, "~", home, 1)
	}

	defaultBasePath := filepath.Join(os.Getenv("HOME"), ".qmlauncher")
	if runtime.GOOS == "windows" {
		home := os.Getenv("USERPROFILE")
		defaultBasePath = filepath.Join(home, ".qmlauncher")
	}

	if customPath == defaultBasePath || customPath == defaultBasePath+"/" || customPath == defaultBasePath+"\\" {
		return defaultPath
	}

	if strings.Contains(customPath, serverUuid) || strings.Contains(customPath, "minecraft") {
		return customPath
	}

	return defaultPath
}

func (m *MinecraftService) fetchVersionManifest() (*VersionManifest, error) {
	log.Printf("[Minecraft] fetchVersionManifest: получение манифеста версий из %s...", MOJANG_VERSION_URL)
	resp, err := http.Get(MOJANG_VERSION_URL)
	if err != nil {
		log.Printf("[Minecraft] fetchVersionManifest: ошибка HTTP запроса: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("[Minecraft] fetchVersionManifest: получен ответ, статус: %d", resp.StatusCode)
	var manifest VersionManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		log.Printf("[Minecraft] fetchVersionManifest: ошибка декодирования JSON: %v", err)
		return nil, err
	}

	log.Printf("[Minecraft] fetchVersionManifest: манифест загружен, версий: %d", len(manifest.Versions))
	return &manifest, nil
}

func (m *MinecraftService) fetchVersionData(url string) (*VersionData, error) {
	log.Printf("[Minecraft] fetchVersionData: загрузка данных версии из %s...", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[Minecraft] fetchVersionData: ошибка HTTP запроса: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("[Minecraft] fetchVersionData: получен ответ, статус: %d", resp.StatusCode)
	var versionData VersionData
	if err := json.NewDecoder(resp.Body).Decode(&versionData); err != nil {
		log.Printf("[Minecraft] fetchVersionData: ошибка декодирования JSON: %v", err)
		return nil, err
	}

	log.Printf("[Minecraft] fetchVersionData: данные версии загружены: MainClass=%s, Libraries=%d, AssetIndex=%s", versionData.MainClass, len(versionData.Libraries), versionData.AssetIndex.ID)
	return &versionData, nil
}

func (m *MinecraftService) downloadFile(url string, destPath string, expectedSHA1 string) error {
	log.Printf("[Minecraft] downloadFile: начало загрузки из %s в %s", url, destPath)

	// Проверяем, существует ли файл и совпадает ли хеш
	if fileInfo, err := os.Stat(destPath); err == nil {
		log.Printf("[Minecraft] downloadFile: файл уже существует: %s (размер: %d байт)", destPath, fileInfo.Size())
		if expectedSHA1 != "" {
			log.Printf("[Minecraft] downloadFile: проверка хеша (ожидается: %s)...", expectedSHA1)
			if actualSHA1, err := m.calculateSHA1(destPath); err == nil && actualSHA1 == expectedSHA1 {
				log.Printf("[Minecraft] downloadFile: хеш совпадает, файл валиден")
				return nil // Файл уже существует и хеш совпадает
			} else if err == nil {
				log.Printf("[Minecraft] downloadFile: хеш не совпадает (получен: %s), перезагружаем", actualSHA1)
			} else {
				log.Printf("[Minecraft] downloadFile: ошибка проверки хеша: %v, перезагружаем", err)
			}
		} else {
			log.Printf("[Minecraft] downloadFile: хеш не указан, файл считается валидным")
			// Если хеш не указан, считаем что файл валиден
			return nil
		}
	}

	// Скачиваем файл
	log.Printf("[Minecraft] downloadFile: отправка HTTP запроса...")
	resp, err := m.followRedirects(url)
	if err != nil {
		log.Printf("[Minecraft] downloadFile: ошибка HTTP запроса: %v", err)
		return err
	}
	defer resp.Body.Close()

	log.Printf("[Minecraft] downloadFile: получен ответ, статус: %d, размер: %d байт", resp.StatusCode, resp.ContentLength)

	// Создаем директорию для файла
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		log.Printf("[Minecraft] downloadFile: ошибка создания директории %s: %v", destDir, err)
		return fmt.Errorf("ошибка создания директории: %v", err)
	}

	file, err := os.Create(destPath)
	if err != nil {
		log.Printf("[Minecraft] downloadFile: ошибка создания файла: %v", err)
		return err
	}
	defer file.Close()

	log.Printf("[Minecraft] downloadFile: копирование данных...")
	bytesWritten, err := io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(destPath)
		log.Printf("[Minecraft] downloadFile: ошибка записи: %v", err)
		return err
	}
	log.Printf("[Minecraft] downloadFile: записано %d байт", bytesWritten)

	// Проверяем хеш
	if expectedSHA1 != "" {
		log.Printf("[Minecraft] downloadFile: проверка хеша (ожидается: %s)...", expectedSHA1)
		actualSHA1, err := m.calculateSHA1(destPath)
		if err != nil {
			os.Remove(destPath)
			log.Printf("[Minecraft] downloadFile: ошибка проверки хеша: %v", err)
			return err
		}
		if actualSHA1 != expectedSHA1 {
			os.Remove(destPath)
			log.Printf("[Minecraft] downloadFile: хеш не совпадает (ожидался: %s, получен: %s)", expectedSHA1, actualSHA1)
			return fmt.Errorf("хеш файла не совпадает: ожидался %s, получен %s", expectedSHA1, actualSHA1)
		}
		log.Printf("[Minecraft] downloadFile: хеш совпадает, файл валиден")
	}

	log.Printf("[Minecraft] downloadFile: загрузка завершена успешно: %s", destPath)
	return nil
}

func (m *MinecraftService) followRedirects(url string) (*http.Response, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Следуем всем редиректам
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (m *MinecraftService) calculateSHA1(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (m *MinecraftService) downloadLibraries(versionData *VersionData, librariesDir string) error {
	log.Printf("[Minecraft] downloadLibraries: начало загрузки библиотек (всего: %d, директория: %s)", len(versionData.Libraries), librariesDir)

	semaphore := make(chan struct{}, CONCURRENCY)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error
	downloadedCount := 0
	skippedCount := 0
	totalLibraries := 0

	for _, lib := range versionData.Libraries {
		// Пропускаем платформенно-специфичные библиотеки, не соответствующие текущей ОС/архитектуре
		nameLower := strings.ToLower(lib.Name)
		osName := runtime.GOOS
		arch := runtime.GOARCH

		hasWindows := strings.Contains(nameLower, "windows")
		hasLinux := strings.Contains(nameLower, "linux")
		hasMac := strings.Contains(nameLower, "osx") || strings.Contains(nameLower, "macos")
		hasArm := strings.Contains(nameLower, "arm64") || strings.Contains(nameLower, "aarch_64") || strings.Contains(nameLower, "aarch64")
		hasX64 := strings.Contains(nameLower, "x86_64") || strings.Contains(nameLower, "x64") || strings.Contains(nameLower, "amd64")

		// Пропускаем чужие ОС
		if hasWindows && osName != "windows" {
			continue
		}
		if hasLinux && osName != "linux" {
			continue
		}
		if hasMac && osName != "darwin" {
			continue
		}

		// Пропускаем чужую архитектуру
		if arch == "amd64" || arch == "x86_64" {
			if hasArm {
				continue
			}
		} else if arch == "arm64" || arch == "aarch64" {
			if hasX64 {
				continue
			}
		}

		// Пропускаем natives библиотеки - они загружаются только через extractNatives для текущей платформы
		// Natives библиотеки имеют classifier в имени (например, "org.lwjgl:lwjgl-opengl:3.3.1:natives-windows")
		if strings.Contains(lib.Name, ":natives-") {
			log.Printf("[Minecraft] downloadLibraries: пропуск natives библиотеки %s (будет загружена через extractNatives для текущей платформы)", lib.Name)
			continue
		}

		// Проверяем правила для библиотеки
		if !m.shouldIncludeLibrary(lib) {
			continue
		}
		totalLibraries++

		wg.Add(1)
		go func(library Library) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			libPath := m.getLibraryPath(library, librariesDir)
			libDir := filepath.Dir(libPath)

			if err := os.MkdirAll(libDir, 0755); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("ошибка создания директории для %s: %v", library.Name, err))
				mu.Unlock()
				return
			}

			// Проверяем существование и хеш
			if fileInfo, err := os.Stat(libPath); err == nil {
				if library.Downloads.Artifact.SHA1 != "" {
					actualSHA1, err := m.calculateSHA1(libPath)
					if err == nil && actualSHA1 == library.Downloads.Artifact.SHA1 {
						mu.Lock()
						skippedCount++
						mu.Unlock()
						return // Файл уже существует и хеш совпадает
					}
				} else if fileInfo.Size() > 0 {
					mu.Lock()
					skippedCount++
					mu.Unlock()
					return // Файл существует и хеш не указан
				}
			}

			// Скачиваем библиотеку
			log.Printf("[Minecraft] downloadLibraries: загрузка %s...", library.Name)
			if err := m.downloadFile(library.Downloads.Artifact.URL, libPath, library.Downloads.Artifact.SHA1); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("ошибка загрузки библиотеки %s: %v", library.Name, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			downloadedCount++
			if downloadedCount%10 == 0 {
				log.Printf("[Minecraft] downloadLibraries: загружено библиотек: %d/%d...", downloadedCount, totalLibraries)
			}
			mu.Unlock()

			// Извлекаем natives если нужно (только для текущей платформы)
			hasClassifiersNatives := false
			for cls := range library.Downloads.Classifiers {
				if strings.HasPrefix(cls, "natives-") {
					hasClassifiersNatives = true
					break
				}
			}
			if library.Natives != nil || hasClassifiersNatives {
				log.Printf("[Minecraft] downloadLibraries: извлечение natives для %s...", library.Name)
				if err := m.extractNatives(library, librariesDir); err != nil {
					mu.Lock()
					errors = append(errors, fmt.Errorf("ошибка извлечения natives для %s: %v", library.Name, err))
					mu.Unlock()
				} else {
					log.Printf("[Minecraft] downloadLibraries: natives извлечены для %s", library.Name)
				}
			}
		}(lib)
	}

	wg.Wait()

	log.Printf("[Minecraft] downloadLibraries: загрузка завершена, загружено: %d, пропущено: %d, ошибок: %d", downloadedCount, skippedCount, len(errors))
	if len(errors) > 0 {
		log.Printf("[Minecraft] downloadLibraries: первая ошибка: %v", errors[0])
		return fmt.Errorf("ошибки при загрузке библиотек: %v", errors[0])
	}

	return nil
}

func (m *MinecraftService) downloadAssets(versionData *VersionData, assetsDir string) error {
	// Загружаем asset index
	assetIndexUrl := versionData.AssetIndex.URL
	assetIndexPath := filepath.Join(assetsDir, "indexes", versionData.AssetIndex.ID+".json")

	if err := os.MkdirAll(filepath.Dir(assetIndexPath), 0755); err != nil {
		return fmt.Errorf("ошибка создания директории для asset index: %v", err)
	}

	if err := m.downloadFile(assetIndexUrl, assetIndexPath, versionData.AssetIndex.SHA1); err != nil {
		return fmt.Errorf("ошибка загрузки asset index: %v", err)
	}

	// Читаем asset index
	assetIndexData, err := os.ReadFile(assetIndexPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения asset index: %v", err)
	}

	var assetIndex AssetIndex
	if err := json.Unmarshal(assetIndexData, &assetIndex); err != nil {
		return fmt.Errorf("ошибка парсинга asset index: %v", err)
	}

	// Загружаем все assets параллельно
	semaphore := make(chan struct{}, ASSET_CONCURRENCY)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error
	downloadedAssets := 0
	skippedAssets := 0

	for key, asset := range assetIndex.Objects {
		wg.Add(1)
		go func(assetKey string, assetInfo AssetInfo) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			hash := assetInfo.Hash
			hashPrefix := hash[:2]
			assetPath := filepath.Join(assetsDir, "objects", hashPrefix, hash)

			// Проверяем существование и хеш
			if _, err := os.Stat(assetPath); err == nil {
				actualSHA1, err := m.calculateSHA1(assetPath)
				if err == nil && actualSHA1 == hash {
					mu.Lock()
					skippedAssets++
					mu.Unlock()
					return // Файл уже существует и хеш совпадает
				}
			}

			// Создаем директорию
			if err := os.MkdirAll(filepath.Dir(assetPath), 0755); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("ошибка создания директории для asset %s: %v", assetKey, err))
				mu.Unlock()
				return
			}

			// Скачиваем asset
			assetUrl := fmt.Sprintf("https://resources.download.minecraft.net/%s/%s", hashPrefix, hash)
			if err := m.downloadFile(assetUrl, assetPath, hash); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("ошибка загрузки asset %s: %v", assetKey, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			downloadedAssets++
			if downloadedAssets%100 == 0 {
				log.Printf("[Minecraft] downloadAssets: загружено ресурсов: %d/%d...", downloadedAssets, len(assetIndex.Objects))
			}
			mu.Unlock()
		}(key, asset)
	}

	wg.Wait()

	log.Printf("[Minecraft] downloadAssets: загрузка завершена, загружено: %d, пропущено: %d, ошибок: %d", downloadedAssets, skippedAssets, len(errors))
	if len(errors) > 0 {
		log.Printf("[Minecraft] downloadAssets: первая ошибка: %v", errors[0])
		return fmt.Errorf("ошибки при загрузке assets: %v", errors[0])
	}

	return nil
}

func (m *MinecraftService) shouldIncludeLibrary(lib Library) bool {
	if len(lib.Rules) == 0 {
		return true
	}

	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}

	for _, rule := range lib.Rules {
		if rule.Action == "disallow" {
			if rule.OS != nil {
				if name, ok := rule.OS["name"].(string); ok && name == osName {
					return false
				}
			}
		} else if rule.Action == "allow" {
			if rule.OS == nil {
				return true
			}
			if name, ok := rule.OS["name"].(string); ok && name == osName {
				return true
			}
		}
	}

	return true
}

func (m *MinecraftService) getLibraryPath(lib Library, librariesDir string) string {
	parts := strings.Split(lib.Name, ":")
	group := strings.ReplaceAll(parts[0], ".", "/")
	artifact := parts[1]
	version := parts[2]

	libPath := filepath.Join(librariesDir, group, artifact, version, artifact+"-"+version+".jar")
	return libPath
}

// extractAllNatives проходит по всем библиотекам и пытается извлечь natives
func (m *MinecraftService) extractAllNatives(versionData *VersionData, librariesDir string) int {
	log.Printf("[Minecraft] extractAllNatives: начало, библиотек всего: %d", len(versionData.Libraries))
	extracted := 0
	processed := 0
	skipped := 0
	for _, lib := range versionData.Libraries {
		nameHasNativeSuffix := strings.Contains(strings.ToLower(lib.Name), ":natives-")
		hasClassifiersNatives := false
		for cls := range lib.Downloads.Classifiers {
			if strings.HasPrefix(cls, "natives-") {
				hasClassifiersNatives = true
				break
			}
		}
		if len(lib.Natives) > 0 || hasClassifiersNatives || nameHasNativeSuffix {
			processed++
			log.Printf("[Minecraft] extractAllNatives: обработка %s (Natives keys=%d, classifiersNatives=%v, nameHasNativeSuffix=%v)", lib.Name, len(lib.Natives), hasClassifiersNatives, nameHasNativeSuffix)
			if err := m.extractNatives(lib, librariesDir); err != nil {
				log.Printf("[Minecraft] ПРЕДУПРЕЖДЕНИЕ: извлечение natives для %s завершилось ошибкой: %v", lib.Name, err)
			} else {
				extracted++
			}
		} else {
			skipped++
		}
	}
	log.Printf("[Minecraft] extractAllNatives: завершено. Обработано: %d, извлечено: %d, пропущено (без natives): %d", processed, extracted, skipped)
	return extracted
}

func (m *MinecraftService) extractNatives(lib Library, librariesDir string) error {
	log.Printf("[Minecraft] extractNatives: начало извлечения natives для библиотеки %s", lib.Name)

	// Если имя библиотеки уже содержит classifier вида :natives-windows[-arch], помечаем это
	var nameClassifier string
	if idx := strings.LastIndex(lib.Name, ":natives-"); idx != -1 {
		nameClassifier = lib.Name[idx+1:] // natives-windows[-arch]
	}

	// Если поле Natives не заполнено, но есть classifiers с префиксом "natives-", строим карту на лету
	if len(lib.Natives) == 0 {
		if lib.Downloads.Classifiers != nil {
			detected := map[string]string{}
			for cls := range lib.Downloads.Classifiers {
				if strings.HasPrefix(cls, "natives-") {
					key := strings.TrimPrefix(cls, "natives-") // windows, windows-x86_64, linux, osx, macos и т.п.
					detected[key] = cls
				}
			}
			if len(detected) > 0 {
				log.Printf("[Minecraft] extractNatives: построена карта natives из classifiers для %s: %v", lib.Name, detected)
				lib.Natives = detected
			}
		}
	}

	if len(lib.Natives) == 0 {
		if nameClassifier == "" {
			log.Printf("[Minecraft] extractNatives: natives не указаны и не найдены в classifiers для библиотеки %s", lib.Name)
			return nil
		}
		// Построим минимальную карту из имени (новый формат, когда classifier в имени и artifact в downloads.artifact)
		lib.Natives = map[string]string{}
		// nameClassifier уже без двоеточия, например "natives-windows" или "natives-windows-x86_64"
		lib.Natives[nameClassifier] = nameClassifier
		log.Printf("[Minecraft] extractNatives: построена карта natives из имени: %v", lib.Natives)
	}

	// Логируем доступные ключи в lib.Natives для отладки
	if len(lib.Natives) > 0 {
		keys := make([]string, 0, len(lib.Natives))
		for k := range lib.Natives {
			keys = append(keys, k)
		}
		log.Printf("[Minecraft] extractNatives: доступные ключи natives для %s: %v", lib.Name, keys)
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH
	minecraftOsName := osName
	if osName == "darwin" {
		minecraftOsName = "osx"
	}

	// Определяем архитектуру для Minecraft natives
	minecraftArch := arch
	if arch == "amd64" || arch == "x86_64" {
		minecraftArch = "x86_64" // Приоритет x86_64 для Windows/Linux
	} else if arch == "arm64" || arch == "aarch64" {
		minecraftArch = "arm64"
	}

	log.Printf("[Minecraft] extractNatives: платформа: %s/%s (minecraft: %s/%s)", osName, arch, minecraftOsName, minecraftArch)

	// Пробуем разные варианты ключей для natives
	// В lib.Natives ключ может быть "windows", "osx", "linux" или "natives-windows", "natives-osx", "natives-linux"
	var nativeClassifier string
	var ok bool

	// Сначала пробуем стандартный ключ
	nativeClassifier, ok = lib.Natives[minecraftOsName]
	if !ok {
		// Пробуем с префиксом "natives-"
		nativeClassifier, ok = lib.Natives["natives-"+minecraftOsName]
	}
	if !ok && osName == "darwin" {
		// Для macOS пробуем "macos"
		nativeClassifier, ok = lib.Natives["macos"]
		if !ok {
			nativeClassifier, ok = lib.Natives["natives-macos"]
		}
	}

	if !ok {
		log.Printf("[Minecraft] extractNatives: нет natives для платформы %s в библиотеке %s", osName, lib.Name)
		return nil // Нет natives для этой ОС
	}
	log.Printf("[Minecraft] extractNatives: найден classifier из Natives: %s", nativeClassifier)

	// Находим native библиотеку
	// В Classifiers ключ map - это classifier (например, "natives-windows" или "natives-windows-x86_64")
	var nativeLib *LibraryDownload
	if lib.Downloads.Classifiers != nil {
		// Сначала пробуем точное совпадение
		if download, found := lib.Downloads.Classifiers[nativeClassifier]; found {
			nativeLib = &download
			log.Printf("[Minecraft] extractNatives: найдена native библиотека по точному совпадению: %s", nativeClassifier)
		} else {
			// Пробуем найти с учетом архитектуры
			// Приоритет: natives-{os}-{arch} > natives-{os}
			// Исключаем неправильные архитектуры (arm64 на x86_64 системе и наоборот)
			var preferredClassifiers []string

			// Формируем список предпочтительных classifiers с учетом архитектуры
			if minecraftArch == "x86_64" || minecraftArch == "amd64" {
				// Для x86_64 приоритет: natives-{os}-x86_64, natives-{os}-x64, natives-{os}
				preferredClassifiers = []string{
					"natives-" + minecraftOsName + "-x86_64",
					"natives-" + minecraftOsName + "-x64",
					"natives-" + minecraftOsName,
					nativeClassifier,
				}
			} else if minecraftArch == "arm64" || minecraftArch == "aarch64" {
				// Для ARM64 приоритет: natives-{os}-arm64, natives-{os}-aarch64, natives-{os}
				preferredClassifiers = []string{
					"natives-" + minecraftOsName + "-arm64",
					"natives-" + minecraftOsName + "-aarch64",
					"natives-" + minecraftOsName,
					nativeClassifier,
				}
			} else {
				// Для других архитектур просто natives-{os}
				preferredClassifiers = []string{
					"natives-" + minecraftOsName,
					nativeClassifier,
				}
			}

			// Ищем по приоритету
			for _, preferred := range preferredClassifiers {
				if download, found := lib.Downloads.Classifiers[preferred]; found {
					nativeLib = &download
					nativeClassifier = preferred
					log.Printf("[Minecraft] extractNatives: найдена native библиотека по приоритету: %s", preferred)
					break
				}
			}

			// Если не нашли по приоритету, ищем по частичному совпадению, исключая неправильные архитектуры
			if nativeLib == nil {
				for classifier, download := range lib.Downloads.Classifiers {
					// Проверяем, содержит ли classifier нужную платформу
					hasPlatform := strings.Contains(classifier, nativeClassifier) ||
						(minecraftOsName == "windows" && strings.Contains(classifier, "windows")) ||
						(minecraftOsName == "osx" && (strings.Contains(classifier, "osx") || strings.Contains(classifier, "macos"))) ||
						(minecraftOsName == "linux" && strings.Contains(classifier, "linux"))

					if !hasPlatform {
						continue
					}

					// Исключаем неправильные архитектуры
					if minecraftArch == "x86_64" || minecraftArch == "amd64" {
						// Исключаем ARM архитектуры
						if strings.Contains(classifier, "arm64") || strings.Contains(classifier, "aarch64") {
							log.Printf("[Minecraft] extractNatives: пропущен classifier с неправильной архитектурой: %s", classifier)
							continue
						}
					} else if minecraftArch == "arm64" || minecraftArch == "aarch64" {
						// Исключаем x86_64 архитектуры
						if strings.Contains(classifier, "x86_64") || strings.Contains(classifier, "x64") || strings.Contains(classifier, "amd64") {
							log.Printf("[Minecraft] extractNatives: пропущен classifier с неправильной архитектурой: %s", classifier)
							continue
						}
					}

					nativeLib = &download
					nativeClassifier = classifier
					log.Printf("[Minecraft] extractNatives: найдена native библиотека по частичному совпадению: %s", classifier)
					break
				}
			}
		}
	}

	// Новый формат: classifier в имени, артефакт в downloads.artifact
	if nativeLib == nil && nameClassifier != "" && lib.Downloads.Artifact.URL != "" {
		nativeClassifier = nameClassifier
		nativeLib = &lib.Downloads.Artifact
		log.Printf("[Minecraft] extractNatives: используем artifact из downloads.artifact для classifier %s", nativeClassifier)
	}

	if nativeLib == nil {
		log.Printf("[Minecraft] extractNatives: native библиотека не найдена для classifier: %s", nativeClassifier)
		return nil
	}
	log.Printf("[Minecraft] extractNatives: native библиотека найдена, URL: %s", nativeLib.URL)

	// Путь к native библиотеке
	var nativePath string
	// Если path указан в artifact, используем его, иначе формируем по имени
	if nativeLib.Path != "" {
		nativePath = filepath.Join(librariesDir, nativeLib.Path)
	} else {
		parts := strings.Split(lib.Name, ":")
		group := strings.ReplaceAll(parts[0], ".", "/")
		artifact := parts[1]
		version := parts[2]
		nativePath = filepath.Join(librariesDir, group, artifact, version, artifact+"-"+version+"-"+nativeClassifier+".jar")
	}
	log.Printf("[Minecraft] extractNatives: путь к native библиотеке: %s", nativePath)

	// Создаем директорию для native библиотеки
	if err := os.MkdirAll(filepath.Dir(nativePath), 0755); err != nil {
		log.Printf("[Minecraft] extractNatives: ошибка создания директории: %v", err)
		return fmt.Errorf("ошибка создания директории для native библиотеки: %v", err)
	}

	// Проверяем, существует ли файл и совпадает ли хеш
	if fileInfo, err := os.Stat(nativePath); err == nil {
		log.Printf("[Minecraft] extractNatives: файл уже существует (размер: %d байт), проверка хеша...", fileInfo.Size())
		if nativeLib.SHA1 != "" {
			actualSHA1, err := m.calculateSHA1(nativePath)
			if err == nil && actualSHA1 == nativeLib.SHA1 {
				log.Printf("[Minecraft] extractNatives: хеш совпадает, продолжаем извлечение")
				// Файл уже существует и хеш совпадает, продолжаем извлечение
			} else if err == nil {
				log.Printf("[Minecraft] extractNatives: хеш не совпадает (ожидался: %s, получен: %s), перезагружаем", nativeLib.SHA1, actualSHA1)
				// Хеш не совпадает, удаляем и скачиваем заново
				os.Remove(nativePath)
			}
		} else if fileInfo.Size() > 0 {
			log.Printf("[Minecraft] extractNatives: хеш не указан, файл существует, продолжаем извлечение")
			// Файл существует и хеш не указан, продолжаем извлечение
		}
	}

	// Скачиваем native библиотеку, если её нет или хеш не совпадает
	if _, err := os.Stat(nativePath); os.IsNotExist(err) {
		log.Printf("[Minecraft] extractNatives: native библиотека не найдена, скачивание из %s...", nativeLib.URL)
		if nativeLib.URL == "" {
			log.Printf("[Minecraft] extractNatives: ошибка - URL для native библиотеки не указан")
			return fmt.Errorf("URL для native библиотеки не указан")
		}
		if err := m.downloadFile(nativeLib.URL, nativePath, nativeLib.SHA1); err != nil {
			log.Printf("[Minecraft] extractNatives: ошибка скачивания native библиотеки: %v", err)
			return fmt.Errorf("ошибка скачивания native библиотеки: %v", err)
		}
		log.Printf("[Minecraft] extractNatives: native библиотека скачана: %s", nativePath)
	} else {
		log.Printf("[Minecraft] extractNatives: native библиотека уже существует: %s", nativePath)
	}

	// Извлекаем native файлы
	log.Printf("[Minecraft] extractNatives: открытие ZIP архива для извлечения...")
	reader, err := zip.OpenReader(nativePath)
	if err != nil {
		log.Printf("[Minecraft] extractNatives: ошибка открытия native библиотеки: %v", err)
		return fmt.Errorf("ошибка открытия native библиотеки: %v", err)
	}
	defer reader.Close()

	nativesDir := filepath.Join(librariesDir, "natives")
	log.Printf("[Minecraft] extractNatives: директория для natives: %s", nativesDir)
	if err := os.MkdirAll(nativesDir, 0755); err != nil {
		log.Printf("[Minecraft] extractNatives: ошибка создания директории natives: %v", err)
		return err
	}

	extractedCount := 0
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".so") || strings.HasSuffix(file.Name, ".dylib") || strings.HasSuffix(file.Name, ".dll") {
			destPath := filepath.Join(nativesDir, filepath.Base(file.Name))
			log.Printf("[Minecraft] extractNatives: извлечение %s в %s...", file.Name, destPath)

			srcFile, err := file.Open()
			if err != nil {
				log.Printf("[Minecraft] extractNatives: ошибка открытия файла в архиве %s: %v", file.Name, err)
				continue
			}

			destFile, err := os.Create(destPath)
			if err != nil {
				srcFile.Close()
				log.Printf("[Minecraft] extractNatives: ошибка создания файла %s: %v", destPath, err)
				continue
			}

			bytesWritten, err := io.Copy(destFile, srcFile)
			srcFile.Close()
			destFile.Close()

			if err != nil {
				log.Printf("[Minecraft] extractNatives: ошибка копирования файла %s: %v", destPath, err)
				continue
			}

			// Делаем исполняемым на Unix
			if runtime.GOOS != "windows" {
				os.Chmod(destPath, 0755)
			}

			extractedCount++
			log.Printf("[Minecraft] extractNatives: извлечен файл %s (%d байт)", destPath, bytesWritten)
		}
	}

	log.Printf("[Minecraft] extractNatives: извлечение завершено, извлечено файлов: %d", extractedCount)

	// Проверяем, что извлечены нужные файлы для Windows
	if runtime.GOOS == "windows" && extractedCount == 0 {
		log.Printf("[Minecraft] extractNatives: ПРЕДУПРЕЖДЕНИЕ - не извлечено ни одного DLL файла для Windows!")
	}

	return nil
}

func (m *MinecraftService) loadVersionJson(path string) (*VersionData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var versionData VersionData
	if err := json.Unmarshal(data, &versionData); err != nil {
		return nil, err
	}

	return &versionData, nil
}

func (m *MinecraftService) buildClasspath(versionData *VersionData, minecraftBasePath string) string {
	var classpath []string
	seenPaths := make(map[string]bool)

	// Добавляем клиент JAR
	versionDir := filepath.Join(minecraftBasePath, "versions", versionData.ID)
	clientJar := filepath.Join(versionDir, versionData.ID+".jar")
	classpath = append(classpath, clientJar)
	seenPaths[clientJar] = true

	// Добавляем библиотеки
	librariesDir := filepath.Join(minecraftBasePath, "libraries")
	addedCount := 0
	skippedCount := 0
	duplicateCount := 0
	for _, lib := range versionData.Libraries {
		if !m.shouldIncludeLibrary(lib) {
			skippedCount++
			continue
		}
		libPath := m.getLibraryPath(lib, librariesDir)
		// Нормализуем путь для сравнения
		normalizedPath, err := filepath.Abs(libPath)
		if err != nil {
			normalizedPath = libPath
		}
		if seenPaths[normalizedPath] {
			duplicateCount++
			continue
		}
		if _, err := os.Stat(libPath); err == nil {
			classpath = append(classpath, libPath)
			seenPaths[normalizedPath] = true
			addedCount++
		} else {
			// Логируем только один раз для каждой отсутствующей библиотеки
			if !seenPaths[libPath+"_notfound"] {
				log.Printf("[Minecraft] buildClasspath: библиотека не найдена: %s (путь: %s)", lib.Name, libPath)
				seenPaths[libPath+"_notfound"] = true
			}
		}
	}

	separator := ":"
	if runtime.GOOS == "windows" {
		separator = ";"
	}

	result := strings.Join(classpath, separator)
	if duplicateCount > 0 {
		log.Printf("[Minecraft] buildClasspath: клиент JAR + %d библиотек (пропущено: %d, дубликатов: %d), длина classpath: %d символов", addedCount, skippedCount, duplicateCount, len(result))
	} else {
		log.Printf("[Minecraft] buildClasspath: клиент JAR + %d библиотек (пропущено: %d), длина classpath: %d символов", addedCount, skippedCount, len(result))
	}
	return result
}

func (m *MinecraftService) buildJVMArgs(versionData *VersionData, settings *Settings, args LaunchMinecraftArgs, minecraftBasePath string, classpathCache string) []string {
	var jvmArgs []string
	seenLibraryPath := false

	// Добавляем память
	jvmArgs = append(jvmArgs, fmt.Sprintf("-Xms%dM", settings.MinMemory))
	jvmArgs = append(jvmArgs, "-Xmx2G")

	// Добавляем пользовательские JVM аргументы
	jvmArgs = append(jvmArgs, settings.JVMArgs...)

	// Обрабатываем аргументы из version.json
	for _, arg := range versionData.Arguments.JVM {
		if str, ok := arg.(string); ok {
			// Пропускаем macOS-специфичные аргументы на Windows
			if runtime.GOOS != "darwin" && str == "-XstartOnFirstThread" {
				continue
			}
			// Отслеживаем -Djava.library.path
			if strings.HasPrefix(str, "-Djava.library.path") {
				seenLibraryPath = true
			}
			// Заменяем переменные
			str = m.replaceVariables(str, versionData, args, minecraftBasePath, classpathCache)
			if str != "" {
				jvmArgs = append(jvmArgs, str)
			}
		} else if rule, ok := arg.(map[string]interface{}); ok {
			// Проверяем правила
			if m.shouldIncludeArgument(rule) {
				if value, ok := rule["value"].(string); ok {
					// Пропускаем macOS-специфичные аргументы на Windows
					if runtime.GOOS != "darwin" && value == "-XstartOnFirstThread" {
						continue
					}
					// Отслеживаем -Djava.library.path
					if strings.HasPrefix(value, "-Djava.library.path") {
						seenLibraryPath = true
					}
					value = m.replaceVariables(value, versionData, args, minecraftBasePath, classpathCache)
					if value != "" {
						jvmArgs = append(jvmArgs, value)
					}
				} else if values, ok := rule["value"].([]interface{}); ok {
					for _, v := range values {
						if str, ok := v.(string); ok {
							// Пропускаем macOS-специфичные аргументы на Windows
							if runtime.GOOS != "darwin" && str == "-XstartOnFirstThread" {
								continue
							}
							// Отслеживаем -Djava.library.path
							if strings.HasPrefix(str, "-Djava.library.path") {
								seenLibraryPath = true
							}
							str = m.replaceVariables(str, versionData, args, minecraftBasePath, classpathCache)
							if str != "" {
								jvmArgs = append(jvmArgs, str)
							}
						}
					}
				}
			}
		}
	}

	// Добавляем natives directory только если его еще нет
	if !seenLibraryPath {
		nativesDir := filepath.Join(minecraftBasePath, "libraries", "natives")
		jvmArgs = append(jvmArgs, "-Djava.library.path="+nativesDir)
	}

	// Специфичные для macOS аргументы
	if runtime.GOOS == "darwin" {
		jvmArgs = append(jvmArgs, "-XstartOnFirstThread")
	}

	return jvmArgs
}

func (m *MinecraftService) buildGameArgs(versionData *VersionData, args LaunchMinecraftArgs, minecraftBasePath string, classpathCache string) []string {
	var gameArgs []string

	// Сначала собираем все аргументы в один список для упрощения обработки
	var allArgs []string
	for _, arg := range versionData.Arguments.Game {
		if str, ok := arg.(string); ok {
			allArgs = append(allArgs, str)
		} else if rule, ok := arg.(map[string]interface{}); ok {
			if m.shouldIncludeGameArgument(rule, args) {
				if value, ok := rule["value"].(string); ok {
					allArgs = append(allArgs, value)
				} else if values, ok := rule["value"].([]interface{}); ok {
					for _, v := range values {
						if str, ok := v.(string); ok {
							allArgs = append(allArgs, str)
						}
					}
				}
			}
		}
	}

	// Теперь обрабатываем аргументы, пропуская quickPlay флаги и их значения
	for i := 0; i < len(allArgs); i++ {
		str := allArgs[i]

		// НЕ пропускаем флаги quickPlay, если они пришли с фронтенда
		// quickPlay флаги из version.json пропускаем, но если они в args.GameArgs - используем их
		// Это позволяет использовать --quickPlayMultiplayer для автоматического подключения
		if len(args.GameArgs) == 0 {
			// Если аргументы приходят из version.json, пропускаем quickPlay флаги
			if str == "--quickPlaySingleplayer" || str == "--quickPlayMultiplayer" || str == "--quickPlayRealms" || str == "--quickPlayPath" {
				// Пропускаем следующий аргумент, если он есть
				if i+1 < len(allArgs) {
					nextStr := m.replaceVariables(allArgs[i+1], versionData, args, minecraftBasePath, classpathCache)
					if nextStr == "" || strings.HasPrefix(nextStr, "${") {
						// Следующий аргумент пустой или переменная - пропускаем оба
						i++ // Пропускаем следующий аргумент
						continue
					}
				}
				// Следующий аргумент не пустой - пропускаем только флаг, следующий будет обработан
				continue
			}
		}

		// Заменяем переменные
		str = m.replaceVariables(str, versionData, args, minecraftBasePath, classpathCache)
		// Пропускаем пустые аргументы и необработанные переменные
		if str != "" && !strings.HasPrefix(str, "${") {
			gameArgs = append(gameArgs, str)
		}
	}

	return gameArgs
}

func (m *MinecraftService) replaceVariables(str string, versionData *VersionData, args LaunchMinecraftArgs, minecraftBasePath string, classpathCache string) string {
	replacements := map[string]string{
		"${natives_directory}": filepath.Join(minecraftBasePath, "libraries", "natives"),
		"${launcher_name}":     "QMLauncher",
		"${launcher_version}":  "1.0.0",
		"${classpath}":         classpathCache,
		"${game_directory}":    minecraftBasePath,
		"${assets_root}":       filepath.Join(minecraftBasePath, "assets"),
		"${assets_index_name}": args.MinecraftVersion,
		"${auth_player_name}":  args.Username,
		"${version_name}":      args.MinecraftVersion,
		"${game_assets}":       filepath.Join(minecraftBasePath, "assets", "virtual", "legacy"),
		"${user_properties}":   "{}",
		"${user_type}":         "legacy",
		"${version_type}":      "release",
		// Quick play variables - remove if not used
		"${quickPlaySingleplayer}": "",
		"${quickPlayMultiplayer}":  "",
		"${quickPlayRealms}":       "",
		"${quickPlayPath}":         "",
	}

	for key, value := range replacements {
		str = strings.ReplaceAll(str, key, value)
	}

	// Remove empty arguments (from quick play variables)
	str = strings.TrimSpace(str)

	return str
}

func (m *MinecraftService) shouldIncludeArgument(rule map[string]interface{}) bool {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}

	if os, ok := rule["os"].(map[string]interface{}); ok {
		if name, ok := os["name"].(string); ok && name != osName {
			return false
		}
	}

	// Проверяем features (например, is_demo_user)
	if features, ok := rule["features"].(map[string]interface{}); ok {
		if isDemo, ok := features["is_demo_user"].(bool); ok && isDemo {
			return false // Исключаем аргументы для demo режима
		}
	}

	return true
}

func (m *MinecraftService) shouldIncludeGameArgument(rule map[string]interface{}, args LaunchMinecraftArgs) bool {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "osx"
	}

	if os, ok := rule["os"].(map[string]interface{}); ok {
		if name, ok := os["name"].(string); ok && name != osName {
			return false
		}
	}

	// Проверяем features (например, is_demo_user)
	if features, ok := rule["features"].(map[string]interface{}); ok {
		if isDemo, ok := features["is_demo_user"].(bool); ok && isDemo {
			return false // Исключаем аргументы для demo режима
		}
	}

	return true
}

// Типы данных

type LaunchMinecraftArgs struct {
	JavaPath         string                 `json:"JavaPath"`
	GameArgs         []string               `json:"GameArgs"`
	JVMArgs          []string               `json:"JVMArgs"`
	WorkingDirectory string                 `json:"WorkingDirectory"`
	MinecraftVersion string                 `json:"MinecraftVersion"`
	HWID             string                 `json:"HWID"`
	LauncherConfig   map[string]interface{} `json:"LauncherConfig"`
	ServerUuid       string                 `json:"ServerUuid"`
	Username         string                 `json:"Username"`
}

type LaunchResult struct {
	Success bool
	Error   string
}

type InstallResult struct {
	Success          bool
	AlreadyInstalled bool
	Message          string
	Error            string
}

type ClientCheckResult struct {
	Success   bool `json:"success"`
	Installed bool `json:"installed"`
	HasClient bool `json:"hasClient"`
}

type UninstallResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type VersionManifest struct {
	Versions []VersionInfo `json:"versions"`
}

type VersionInfo struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type VersionData struct {
	ID         string         `json:"id"`
	MainClass  string         `json:"mainClass"`
	Arguments  Arguments      `json:"arguments"`
	Libraries  []Library      `json:"libraries"`
	AssetIndex AssetIndexInfo `json:"assetIndex"`
	Downloads  Downloads      `json:"downloads"`
}

type Arguments struct {
	Game []interface{} `json:"game"`
	JVM  []interface{} `json:"jvm"`
}

type Library struct {
	Name      string            `json:"name"`
	Rules     []LibraryRule     `json:"rules,omitempty"`
	Downloads LibraryDownloads  `json:"downloads"`
	Natives   map[string]string `json:"natives,omitempty"`
}

type LibraryRule struct {
	Action string                 `json:"action"`
	OS     map[string]interface{} `json:"os,omitempty"`
}

type LibraryDownloads struct {
	Artifact    LibraryDownload            `json:"artifact"`
	Classifiers map[string]LibraryDownload `json:"classifiers,omitempty"`
}

type LibraryDownload struct {
	URL        string `json:"url"`
	SHA1       string `json:"sha1"`
	Size       int    `json:"size"`
	Classifier string `json:"-"`
	Path       string `json:"path,omitempty"`
}

type AssetIndexInfo struct {
	ID   string `json:"id"`
	SHA1 string `json:"sha1"`
	Size int    `json:"size"`
	URL  string `json:"url"`
}

type AssetIndex struct {
	Objects map[string]AssetInfo `json:"objects"`
}

type AssetInfo struct {
	Hash string `json:"hash"`
	Size int    `json:"size"`
}

type Downloads struct {
	Client DownloadInfo `json:"client"`
}

type DownloadInfo struct {
	SHA1 string `json:"sha1"`
	Size int    `json:"size"`
	URL  string `json:"url"`
}

// setCmdHideWindow скрывает консольное окно при запуске процесса
// На Windows использует SysProcAttr для скрытия окна через CREATE_NO_WINDOW флаг
// На других платформах ничего не делает
func setCmdHideWindow(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		// CREATE_NO_WINDOW (0x08000000) предотвращает всплытие консоли
		// Используем рефлексию для установки CreationFlags, так как это поле
		// доступно только на Windows и может отсутствовать в определении типа на других платформах
		attr := &syscall.SysProcAttr{}
		v := reflect.ValueOf(attr).Elem()

		// Пытаемся установить CreationFlags через рефлексию
		if f := v.FieldByName("CreationFlags"); f.IsValid() && f.CanSet() {
			f.SetUint(0x08000000) // CREATE_NO_WINDOW
		}

		// Пытаемся установить HideWindow, если доступно
		if f := v.FieldByName("HideWindow"); f.IsValid() && f.CanSet() {
			f.SetBool(true)
		}

		cmd.SysProcAttr = attr
	}
}
