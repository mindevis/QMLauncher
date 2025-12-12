package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DEFAULT_JAVA_VERSION = "17"
	ADOPTIUM_API_URL     = "https://api.adoptium.net/v3/assets/latest/%s/hotspot"
	AZUL_API_URL         = "https://api.azul.com/metadata/v1/zulu/packages"
)

type JavaService struct {
	app *App
}

func NewJavaService(app *App) *JavaService {
	return &JavaService{app: app}
}

// InstallJava устанавливает Java
func (j *JavaService) InstallJava(vendor string, version string, serverUuid string) error {
	log.Printf("[Java] Начало установки Java: vendor=%s, version=%s, serverUuid=%s", vendor, version, serverUuid)

	settings, err := j.app.GetSettings()
	if err != nil {
		log.Printf("[Java] Ошибка получения настроек: %v", err)
		return fmt.Errorf("ошибка получения настроек: %v", err)
	}

	serverUuidForPath := serverUuid
	if serverUuidForPath == "" {
		serverUuidForPath = "global"
	}
	log.Printf("[Java] Используется serverUuid для пути: %s", serverUuidForPath)

	// Определяем путь установки Java
	javaBasePath := j.getJavaBasePath(settings, serverUuidForPath)
	javaDir := filepath.Join(javaBasePath, vendor, version)
	log.Printf("[Java] Путь установки: %s", javaDir)

	// Проверяем, не установлена ли уже Java
	javaPath := j.findJavaExecutable(javaDir)
	if javaPath != "" {
		log.Printf("[Java] Найдена существующая установка Java: %s", javaPath)
		// Проверяем архитектуру
		if err := j.validateJavaArchitecture(javaPath); err != nil {
			log.Printf("[Java] Неправильная архитектура Java, удаляем: %v", err)
			// Удаляем неправильную версию
			os.RemoveAll(javaDir)
		} else {
			log.Printf("[Java] Java уже установлена и валидна: %s", javaPath)
			return nil // Java уже установлена
		}
	} else {
		log.Printf("[Java] Java не найдена, начинаем установку")
	}

	// Определяем архитектуру
	arch := runtime.GOARCH
	osName := runtime.GOOS
	log.Printf("[Java] Платформа: %s/%s", osName, arch)

	// Скачиваем и устанавливаем Java
	var downloadUrl string

	if vendor == "openjdk" || vendor == "adoptium" {
		log.Printf("[Java] Получение URL для Adoptium...")
		var err error
		downloadUrl, err = j.getAdoptiumDownloadUrl(version, osName, arch)
		if err != nil {
			log.Printf("[Java] Ошибка получения URL Adoptium: %v", err)
			return fmt.Errorf("ошибка получения URL для скачивания: %v", err)
		}
		log.Printf("[Java] URL Adoptium получен: %s", downloadUrl)
	} else if vendor == "azul" {
		log.Printf("[Java] Получение URL для Azul...")
		var err error
		downloadUrl, err = j.getAzulDownloadUrl(version, osName, arch)
		if err != nil {
			log.Printf("[Java] Ошибка получения URL Azul: %v", err)
			return fmt.Errorf("ошибка получения URL для скачивания: %v", err)
		}
		log.Printf("[Java] URL Azul получен: %s", downloadUrl)
	} else {
		log.Printf("[Java] Неподдерживаемый поставщик: %s", vendor)
		return fmt.Errorf("неподдерживаемый поставщик Java: %s", vendor)
	}

	// Создаем директорию для скачивания, если её нет
	log.Printf("[Java] Создание директории для скачивания: %s", javaBasePath)
	if err := os.MkdirAll(javaBasePath, 0755); err != nil {
		log.Printf("[Java] Ошибка создания директории: %v", err)
		return fmt.Errorf("ошибка создания директории для Java: %v", err)
	}

	// Скачиваем Java во временный файл, затем переименовываем
	tempZipPath := filepath.Join(javaBasePath, "java-download.tmp")
	zipPath := filepath.Join(javaBasePath, "java-download.zip")

	// Удаляем старые временные файлы если они есть
	os.Remove(tempZipPath)
	os.Remove(zipPath)

	log.Printf("[Java] Начало скачивания Java из %s во временный файл %s", downloadUrl, tempZipPath)
	if err := j.downloadFile(downloadUrl, tempZipPath); err != nil {
		log.Printf("[Java] Ошибка скачивания: %v", err)
		os.Remove(tempZipPath) // Удаляем неполный файл
		return fmt.Errorf("ошибка скачивания Java: %v", err)
	}

	// Проверяем размер файла
	fileInfo, err := os.Stat(tempZipPath)
	if err != nil {
		log.Printf("[Java] Ошибка проверки размера файла: %v", err)
		os.Remove(tempZipPath)
		return fmt.Errorf("ошибка проверки скачанного файла: %v", err)
	}
	if fileInfo.Size() == 0 {
		log.Printf("[Java] Ошибка: скачанный файл имеет нулевой размер")
		os.Remove(tempZipPath)
		return fmt.Errorf("скачанный файл имеет нулевой размер")
	}
	log.Printf("[Java] Скачивание завершено: %s (размер: %d байт)", tempZipPath, fileInfo.Size())

	// Переименовываем временный файл в финальный
	if err := os.Rename(tempZipPath, zipPath); err != nil {
		log.Printf("[Java] Ошибка переименования файла: %v", err)
		os.Remove(tempZipPath)
		return fmt.Errorf("ошибка переименования скачанного файла: %v", err)
	}

	// Удаляем архив только после успешной распаковки
	defer func() {
		if err := os.Remove(zipPath); err != nil {
			log.Printf("[Java] Предупреждение: не удалось удалить архив: %v", err)
		}
	}()

	// Создаем директорию для распаковки, если её нет
	log.Printf("[Java] Создание директории для распаковки: %s", javaDir)
	if err := os.MkdirAll(javaDir, 0755); err != nil {
		log.Printf("[Java] Ошибка создания директории для распаковки: %v", err)
		return fmt.Errorf("ошибка создания директории для распаковки Java: %v", err)
	}

	// Распаковываем Java
	log.Printf("[Java] Начало распаковки Java из %s в %s", zipPath, javaDir)
	if err := j.extractJava(zipPath, javaDir); err != nil {
		log.Printf("[Java] Ошибка распаковки: %v", err)
		return fmt.Errorf("ошибка распаковки Java: %v", err)
	}
	log.Printf("[Java] Распаковка завершена")

	// Проверяем установку
	finalJavaPath := j.findJavaExecutable(javaDir)
	if finalJavaPath != "" {
		log.Printf("[Java] Установка Java завершена успешно: %s", finalJavaPath)
	} else {
		log.Printf("[Java] Предупреждение: Java распакована, но исполняемый файл не найден")
	}

	return nil
}

// GetJavaPath возвращает путь к Java
func (j *JavaService) GetJavaPath(serverUuid string) (string, error) {
	settings, err := j.app.GetSettings()
	if err != nil {
		return "", fmt.Errorf("ошибка получения настроек: %v", err)
	}

	// Если указан пользовательский путь, используем его
	if settings.JavaPath != "" {
		javaPath := strings.Replace(settings.JavaPath, "~", os.Getenv("HOME"), 1)
		if runtime.GOOS == "windows" {
			home := os.Getenv("USERPROFILE")
			javaPath = strings.Replace(javaPath, "~", home, 1)
		}

		if _, err := os.Stat(javaPath); err == nil {
			// Проверяем, что это Java
			if j.isJavaExecutable(javaPath) {
				return javaPath, nil
			}
		}
	}

	// Ищем Java в стандартных местах
	serverUuidForPath := serverUuid
	if serverUuidForPath == "" {
		serverUuidForPath = "global"
	}

	javaBasePath := j.getJavaBasePath(settings, serverUuidForPath)

	// Пробуем разные версии и поставщиков
	vendors := []string{"openjdk", "adoptium", "azul"}
	versions := []string{"17", "21", "11", "8"}

	for _, vendor := range vendors {
		for _, version := range versions {
			javaDir := filepath.Join(javaBasePath, vendor, version)
			javaPath := j.findJavaExecutable(javaDir)
			if javaPath != "" {
				return javaPath, nil
			}
		}
	}

	return "", fmt.Errorf("Java не найдена")
}

// ValidateJavaPath проверяет путь к Java
func (j *JavaService) ValidateJavaPath(javaPath string) (*JavaValidationResult, error) {
	if javaPath == "" {
		return &JavaValidationResult{Valid: false, Error: "Путь к Java не указан"}, nil
	}

	if _, err := os.Stat(javaPath); os.IsNotExist(err) {
		return &JavaValidationResult{Valid: false, Error: "Java не найдена по указанному пути"}, nil
	}

	if !j.isJavaExecutable(javaPath) {
		return &JavaValidationResult{Valid: false, Error: "Указанный путь не является исполняемым файлом Java"}, nil
	}

	// Проверяем версию Java
	version, err := j.getJavaVersion(javaPath)
	if err != nil {
		return &JavaValidationResult{Valid: false, Error: fmt.Sprintf("Ошибка получения версии Java: %v", err)}, nil
	}

	return &JavaValidationResult{Valid: true, Version: version}, nil
}

// Вспомогательные методы

func (j *JavaService) getJavaBasePath(settings *Settings, serverUuid string) string {
	// Если указан пользовательский путь, используем его базовую директорию
	if settings.JavaPath != "" {
		javaPath := strings.Replace(settings.JavaPath, "~", os.Getenv("HOME"), 1)
		if runtime.GOOS == "windows" {
			home := os.Getenv("USERPROFILE")
			javaPath = strings.Replace(javaPath, "~", home, 1)
		}

		// Проверяем, содержит ли путь server_uuid или vendor/version структуру
		if strings.Contains(javaPath, serverUuid) || strings.Contains(javaPath, "openjdk") || strings.Contains(javaPath, "azul") {
			return filepath.Dir(filepath.Dir(filepath.Dir(javaPath))) // Поднимаемся на 3 уровня
		}
	}

	// Используем структуру с server_uuid
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}

	return filepath.Join(home, ".qmlauncher", serverUuid, "java")
}

func (j *JavaService) findJavaExecutable(javaDir string) string {
	javaBinName := "java"
	if runtime.GOOS == "windows" {
		javaBinName = "java.exe"
	}

	// Ищем в стандартных местах
	possiblePaths := []string{
		filepath.Join(javaDir, "bin", javaBinName),
		filepath.Join(javaDir, javaBinName),
	}

	// Рекурсивно ищем в директории
	filepath.Walk(javaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == javaBinName {
			possiblePaths = append(possiblePaths, path)
		}
		return nil
	})

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func (j *JavaService) isJavaExecutable(path string) bool {
	javaBinName := "java"
	if runtime.GOOS == "windows" {
		javaBinName = "java.exe"
	}

	return strings.HasSuffix(path, javaBinName) || strings.HasSuffix(path, "java")
}

func (j *JavaService) getJavaVersion(javaPath string) (string, error) {
	cmd := exec.Command(javaPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	outputStr := string(output)
	// Парсим версию из вывода
	// Формат обычно: "openjdk version "17.0.1" 2021-10-19"
	parts := strings.Fields(outputStr)
	for i, part := range parts {
		if part == "version" && i+1 < len(parts) {
			version := strings.Trim(parts[i+1], "\"")
			return version, nil
		}
	}

	return "unknown", nil
}

func (j *JavaService) validateJavaArchitecture(javaPath string) error {
	cmd := exec.Command(javaPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	outputStr := string(output)
	arch := runtime.GOARCH

	// Проверяем архитектуру в выводе
	if arch == "amd64" || arch == "x86_64" {
		if strings.Contains(outputStr, "aarch64") || strings.Contains(outputStr, "arm64") {
			return fmt.Errorf("неправильная архитектура: ожидается x86_64, получена ARM")
		}
	}

	return nil
}

func (j *JavaService) getAdoptiumDownloadUrl(version string, osName string, arch string) (string, error) {
	// Преобразуем архитектуру для API
	apiArch := arch
	if arch == "amd64" || arch == "x86_64" {
		apiArch = "x64"
	}

	apiOs := osName
	if osName == "darwin" {
		apiOs = "mac"
	}

	url := fmt.Sprintf(ADOPTIUM_API_URL, version)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var releases []AdoptiumRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}

	// Ищем подходящий релиз: только полноценный JDK (исключаем debug/test релизы), без ARM
	for _, release := range releases {
		if release.Binary.OS != apiOs || release.Binary.Architecture != apiArch {
			continue
		}
		// Оставляем только image_type=jdk (не jre, не debugimage, не testimage)
		if release.Binary.ImageType != "jdk" {
			continue
		}
		// Исключаем ARM версии
		if strings.Contains(release.Binary.ImageType, "aarch64") || strings.Contains(release.Binary.ImageType, "arm64") {
			continue
		}
		// Полностью исключаем debug и test релизы по ссылке (проверяем все возможные варианты)
		linkLower := strings.ToLower(release.Binary.Package.Link)
		if strings.Contains(linkLower, "debug") ||
			strings.Contains(linkLower, "test") ||
			strings.Contains(linkLower, "debugimage") ||
			strings.Contains(linkLower, "testimage") ||
			strings.Contains(linkLower, "debug-image") ||
			strings.Contains(linkLower, "test-image") {
			log.Printf("[Java] Пропущен debug/test релиз: %s", release.Binary.Package.Link)
			continue
		}
		log.Printf("[Java] Выбран подходящий JDK релиз: %s", release.Binary.Package.Link)
		return release.Binary.Package.Link, nil
	}

	return "", fmt.Errorf("не найден подходящий релиз для %s/%s", osName, arch)
}

func (j *JavaService) getAzulDownloadUrl(version string, osName string, arch string) (string, error) {
	// Azul API требует другого подхода
	// Упрощенная версия - используем прямые ссылки
	apiOs := osName
	if osName == "darwin" {
		apiOs = "macos"
	}

	apiArch := arch
	if arch == "amd64" || arch == "x86_64" {
		apiArch = "x64"
	}

	// Формируем URL для Azul
	url := fmt.Sprintf("https://cdn.azul.com/zulu/bin/zulu%s-ca-jdk%s-%s_%s.zip", version, version, apiOs, apiArch)

	// Проверяем доступность
	resp, err := http.Head(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return url, nil
	}

	return "", fmt.Errorf("не найден подходящий релиз Azul для %s/%s", osName, arch)
}

func (j *JavaService) downloadFile(url string, destPath string) error {
	// Создаем директорию для файла, если её нет
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %v", destDir, err)
	}

	// Удаляем файл если он существует (на случай повторной попытки)
	os.Remove(destPath)

	client := &http.Client{
		Timeout: 0, // Без таймаута для больших файлов
	}

	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("[Java] downloadFile: попытка %d/%d отправка HTTP запроса к %s", attempt, maxAttempts, url)
		resp, err := client.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("ошибка HTTP запроса: %v", err)
			log.Printf("[Java] downloadFile: ошибка HTTP запроса: %v", err)
		} else {
			func() {
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					lastErr = fmt.Errorf("ошибка скачивания: статус %d", resp.StatusCode)
					log.Printf("[Java] downloadFile: неожиданный статус код: %d", resp.StatusCode)
					return
				}

				// Получаем размер файла для логирования
				contentLength := resp.ContentLength
				if contentLength > 0 {
					log.Printf("[Java] downloadFile: ожидаемый размер файла: %d байт", contentLength)
				}

				file, err := os.Create(destPath)
				if err != nil {
					lastErr = fmt.Errorf("ошибка создания файла %s: %v", destPath, err)
					log.Printf("[Java] downloadFile: ошибка создания файла: %v", err)
					return
				}
				defer func() {
					if closeErr := file.Close(); closeErr != nil {
						log.Printf("[Java] downloadFile: ошибка закрытия файла: %v", closeErr)
					}
				}()

				log.Printf("[Java] downloadFile: начало записи данных...")
				written, err := io.Copy(file, resp.Body)
				if err != nil {
					lastErr = fmt.Errorf("ошибка записи файла: %v", err)
					log.Printf("[Java] downloadFile: ошибка записи файла: %v (записано: %d байт)", err, written)
					file.Close()
					os.Remove(destPath) // Удаляем неполный файл
					return
				}

				log.Printf("[Java] downloadFile: записано %d байт", written)
				if contentLength > 0 && written != contentLength {
					log.Printf("[Java] downloadFile: предупреждение: записано %d байт, ожидалось %d байт", written, contentLength)
				}

				// Синхронизируем файл на диск
				if err := file.Sync(); err != nil {
					log.Printf("[Java] downloadFile: предупреждение: ошибка синхронизации файла: %v", err)
				}

				lastErr = nil // успешная попытка
			}()
		}

		if lastErr == nil {
			return nil
		}

		// Если не последний заход — подождать и повторить
		if attempt < maxAttempts {
			backoff := time.Duration(attempt) * 2 * time.Second
			log.Printf("[Java] downloadFile: повтор через %v...", backoff)
			time.Sleep(backoff)
			// На повторе удаляем возможный неполный файл
			os.Remove(destPath)
		}
	}

	return lastErr
}

func (j *JavaService) extractJava(zipPath string, destDir string) error {
	log.Printf("[Java] extractJava: начало распаковки из %s в %s", zipPath, destDir)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		log.Printf("[Java] extractJava: ошибка открытия ZIP: %v", err)
		return err
	}
	defer reader.Close()

	log.Printf("[Java] extractJava: ZIP открыт, файлов в архиве: %d", len(reader.File))

	// Находим корневую директорию (может быть вложенной)
	var rootDir string
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "/bin/java") || strings.HasSuffix(file.Name, "/bin/java.exe") {
			parts := strings.Split(file.Name, "/")
			if len(parts) > 1 {
				rootDir = strings.Join(parts[:len(parts)-2], "/")
				break
			}
		}
	}
	if rootDir != "" {
		log.Printf("[Java] extractJava: найдена корневая директория: %s", rootDir)
	} else {
		log.Printf("[Java] extractJava: корневая директория не найдена, используем корень архива")
	}

	// Извлекаем файлы
	filesExtracted := 0
	dirsCreated := 0
	for _, file := range reader.File {
		// Пропускаем корневую директорию если она есть
		destPath := filepath.Join(destDir, strings.TrimPrefix(file.Name, rootDir+"/"))

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				log.Printf("[Java] extractJava: ошибка создания директории %s: %v", destPath, err)
				return err
			}
			dirsCreated++
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			log.Printf("[Java] extractJava: ошибка создания родительской директории для %s: %v", destPath, err)
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			log.Printf("[Java] extractJava: ошибка открытия файла в архиве %s: %v", file.Name, err)
			return err
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			srcFile.Close()
			log.Printf("[Java] extractJava: ошибка создания файла %s: %v", destPath, err)
			return err
		}

		_, err = io.Copy(destFile, srcFile)
		srcFile.Close()
		destFile.Close()

		if err != nil {
			log.Printf("[Java] extractJava: ошибка копирования файла %s: %v", destPath, err)
			return err
		}

		// Делаем исполняемым на Unix
		if runtime.GOOS != "windows" {
			os.Chmod(destPath, 0755)
		}

		filesExtracted++
		if filesExtracted%100 == 0 {
			log.Printf("[Java] extractJava: извлечено файлов: %d...", filesExtracted)
		}
	}

	log.Printf("[Java] extractJava: распаковка завершена, извлечено файлов: %d, создано директорий: %d", filesExtracted, dirsCreated)
	return nil
}

// Типы данных

type JavaValidationResult struct {
	Valid   bool
	Version string
	Error   string
}

type AdoptiumRelease struct {
	Binary AdoptiumBinary `json:"binary"`
}

type AdoptiumBinary struct {
	OS           string          `json:"os"`
	Architecture string          `json:"architecture"`
	ImageType    string          `json:"image_type"`
	Package      AdoptiumPackage `json:"package"`
}

type AdoptiumPackage struct {
	Link string `json:"link"`
}
