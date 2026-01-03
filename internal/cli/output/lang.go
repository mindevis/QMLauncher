package output

import "golang.org/x/text/language"

type translations map[string]string

var en = translations{
	"instance":    "Manage Minecraft instances",
	"auth":        "Manage account authentication",
	"java":        "Manage Java runtime installations",
	"about":       "Display launcher version and about",
	"update":      "Manage application updates",
	"list":        "List all instances",
	"java_list":   "List all installed Java versions",
	"completions": "Outputs shell command to install completions",

	"login":               "Login in to an account",
	"login.code.fetching": "Loading device code...",
	"login.code":          "Use the code %s at %s to sign in",
	"login.browser":       "A web browser will be opened to continue authenticatication.",
	"login.url":           "If the browser does not open, please copy and paste this URL into your browser: %s",
	"login.complete":      "Logged in as %s",
	"login.redirect":      "Logged in! You can close this window and return to the launcher.",
	"login.redirectfail":  "Failed to log in: An error occurred during authentication.",
	"login.arg.nobrowser": "Use device code instead of browser for authentication",

	"logout":          "Log out of an account",
	"logout.complete": "Logged out from account.",

	"create":                   "Create a new instance",
	"create.complete":          "Created instance '%s' with Minecraft %s (%s%s)",
	"create.arg.id":            "Instance name",
	"create.arg.loader":        "Mod loader",
	"create.arg.version":       "Game version",
	"create.arg.loaderversion": "Mod loader version",

	"delete":          "Delete an instance",
	"delete.confirm":  "Are you sure you want to delete this instance?",
	"delete.warning":  "'%s' will be lost forever (a long time!) [y/n] ",
	"delete.complete": "Deleted instance '%s'",
	"delete.abort":    "Operation aborted",
	"delete.arg.id":   "Instance to delete",
	"delete.arg.yes":  "Assume yes to all questions",

	"export":            "Export an instance to ZIP archive",
	"export.arg.id":     "Instance to export",
	"export.arg.output": "Output archive path",
	"export.arg.update": "Update existing export file",

	"import":           "Import an instance from ZIP archive",
	"import.arg.path":  "Path to ZIP archive",
	"import.arg.name":  "Name for imported instance",
	"import.arg.force": "Overwrite existing instance",
	"import.arg.merge": "Merge with existing instance (add missing files only)",

	"update_check":    "Check for available updates",
	"update_download": "Download and install available updates",
	"update_info":     "Show current version information",

	"list_exports":          "List exported instance archives",
	"list_exports_arg_path": "Directory to search for exports",

	"mods":        "List mods in an instance",
	"mods.arg.id": "Instance to list mods for",
	"mods.empty":  "No mods found in this instance",

	"resourcepacks":        "List resource packs in an instance",
	"resourcepacks.arg.id": "Instance to list resource packs for",
	"resourcepacks.empty":  "No resource packs found in this instance",

	"shaders":        "List shader packs in an instance",
	"shaders.arg.id": "Instance to list shader packs for",
	"shaders.empty":  "No shader packs found in this instance",

	"mods.table.name":       "Name",
	"mods.table.curseforge": "CurseForge",
	"mods.table.modrinth":   "Modrinth",
	"mods.table.size":       "Size",

	"resourcepacks.table.name":       "Name",
	"resourcepacks.table.curseforge": "CurseForge",
	"resourcepacks.table.modrinth":   "Modrinth",
	"resourcepacks.table.size":       "Size",
	"shaders.table.name":             "Name",
	"shaders.table.size":             "Size",

	"rename":          "Rename an instance",
	"rename.complete": "Renamed instance.",
	"rename.arg.id":   "Instance to rename",
	"rename.arg.new":  "New name for instance",

	"search":               "Search versions",
	"search.complete":      "Found %d entries",
	"search.table.version": "Version",
	"search.table.type":    "Type",
	"search.table.date":    "Release Date",
	"search.table.name":    "Name",
	"instance.table.path":  "Path",
	"java.table.name":      "Name",
	"java.table.path":      "Path",
	"search.arg.query":     "Search query",
	"search.arg.kind":      "What to search for",
	"search.arg.reverse":   "Reverse the listing",

	"start":                    "Start the specified instance",
	"start.arg.id":             "Instance to launch",
	"start.arg.username":       "Set username (offline mode)",
	"start.arg.server":         "Join a server upon starting the game",
	"start.arg.world":          "Join a world upon starting the game",
	"start.arg.demo":           "Start the game in demo mode",
	"start.arg.disablemp":      "Disable multiplayer",
	"start.arg.disablechat":    "Disable chat",
	"start.arg.width":          "Game window width",
	"start.arg.height":         "Game window height",
	"start.arg.jvm":            "Path to the JVM",
	"start.arg.jvmargs":        "Extra JVM arguments",
	"start.arg.minmemory":      "Minimum memory",
	"start.arg.maxmemory":      "Maximum memory",
	"start.arg.nojavawindow":   "Use javaw.exe instead of java.exe on Windows (no console window)",
	"start.arg.prepare":        "Install all necessary resources but do not start the game.",
	"start.arg.opts":           "Game Options",
	"start.arg.overrides":      "Configuration Overrides",
	"start.prepared":           "Game prepared successfully.",
	"start.processing":         "Post processors are being run. This may take some time.",
	"start.launch.downloading": "Downloading files",
	"start.launch.assets":      "Identified %d assets",
	"start.launch.libraries":   "Identified %d libraries",
	"start.launch.metadata":    "Version metadata retrieved",
	"start.launch.jvmargs":     "JVM arguments: %s",
	"start.launch.gameargs":    "Game arguments: %s",
	"start.launch.info":        "Starting main class %q. Game directory is %q.",
	"start.launch":             "Launching game as %s",

	"arg.verbosity": "Increase launcher output verbosity",
	"arg.dir":       "Root directory for launcher files",
	"arg.nocolor":   "Disable all color output. The NO_COLOR environment variable is also supported.",
	"arg.lang":      "Language for output",

	"tip.internet":  "Check your internet connection.",
	"tip.cache":     "Remote resources were not cached and were unable to be retrieved. Check your Internet connection.",
	"tip.configure": "Configure this instance with the `instance.toml` file within the instance directory.",
	"tip.nojvm":     "If a Mojang-provided JVM is not available, you can install it yourself and set the path to the Java executable in the instance configuration.",
	"tip.noaccount": "To launch in offline mode, use the --username (-u) flag.",

	"launcher.description": "A minimal command-line Minecraft launcher.",
	"launcher.license":     "Licensed MIT",
	"launcher.copyright":   "Copyright 2024-2025 telecter",
	"launcher.error":       "Error",
	"launcher.warning":     "Warning",
	"launcher.debug":       "Debug",
	"launcher.tip":         "Tip",
	"cli.title":            "QMLauncher CLI",
	"cli.subtitle":         "Minecraft launcher with mod support",
	"cli.usage":            "USAGE:",
	"cli.usage.cmd":        "  QMLauncher [command]",
	"cli.commands":         "AVAILABLE COMMANDS:",
	"cli.cmd.instance":     "  instance, -i         Manage Minecraft instances",
	"cli.cmd.update":       "  update               Update the launcher",
	"cli.cmd.auth":         "  auth                 Manage authentication",
	"cli.cmd.search":       "  search               Search for Minecraft versions",
	"cli.cmd.java":         "  java                 Manage Java installations",
	"cli.cmd.about":        "  about                Show version information",
	"cli.aliases":          "ALIASES:",
	"cli.alias.i":          "  -i                   Shortcut for 'instance' command",
	"cli.alias.s":          "  -s                   Shortcut for 'start' command (in instance context)",
	"cli.alias.is":         "  -is                  Shortcut for 'instance start'",
	"cli.help":             "Use 'QMLauncher [command] --help' for more information about a command.",
}

var ru = translations{
	"instance":    "Управление инстансами Minecraft",
	"auth":        "Управление аутентификацией аккаунта",
	"java":        "Управление установками Java runtime",
	"about":       "Показать версию лаунчера и информацию",
	"update":      "Управление обновлениями приложения",
	"list":        "Показать все инстансы",
	"java_list":   "Показать все установленные версии Java",
	"completions": "Вывести команду для установки автодополнения",

	"login":               "Войти в аккаунт",
	"login.code.fetching": "Загрузка кода устройства...",
	"login.code":          "Используйте код %s на %s для входа",
	"login.browser":       "Будет открыт веб-браузер для продолжения аутентификации.",
	"login.url":           "Если браузер не открылся, скопируйте и вставьте этот URL в браузер: %s",
	"login.complete":      "Вход выполнен как %s",
	"login.redirect":      "Вход выполнен! Вы можете закрыть это окно и вернуться в лаунчер.",
	"login.redirectfail":  "Не удалось войти: Произошла ошибка при аутентификации.",
	"login.arg.nobrowser": "Использовать код устройства вместо браузера для аутентификации",

	"logout":          "Выйти из аккаунта",
	"logout.complete": "Выход из аккаунта выполнен.",

	"create":                   "Создать новый инстанс",
	"create.complete":          "Создан инстанс '%s' с Minecraft %s (%s%s)",
	"create.arg.id":            "Имя инстанса",
	"create.arg.loader":        "Мод лоадер",
	"create.arg.version":       "Версия игры",
	"create.arg.loaderversion": "Версия мод лоадера",

	"delete":          "Удалить инстанс",
	"delete.confirm":  "Вы уверены, что хотите удалить этот инстанс?",
	"delete.warning":  "'%s' будет потерян навсегда (долго!) [y/n] ",
	"delete.complete": "Инстанс '%s' удален",
	"delete.abort":    "Операция отменена",
	"delete.arg.id":   "Инстанс для удаления",
	"delete.arg.yes":  "Отвечать 'да' на все вопросы",

	"export":            "Экспортировать инстанс в ZIP архив",
	"export.arg.id":     "Инстанс для экспорта",
	"export.arg.output": "Путь для сохранения архива",
	"export.arg.update": "Обновить существующий файл экспорта",

	"import":           "Импортировать инстанс из ZIP архива",
	"import.arg.path":  "Путь к ZIP файлу",
	"import.arg.name":  "Имя для импортированного инстанса",
	"import.arg.force": "Перезаписать существующий инстанс",
	"import.arg.merge": "Объединить с существующим инстансом (добавить только отсутствующие файлы)",

	"update_check":    "Проверить наличие обновлений",
	"update_download": "Скачать и установить доступные обновления",
	"update_info":     "Показать текущую информацию о версии",

	"list_exports":          "Показать экспортированные архивы инстансов",
	"list_exports_arg_path": "Директория для поиска экспортов",

	"mods":        "Показать моды в инстансе",
	"mods.arg.id": "Инстанс для показа модов",
	"mods.empty":  "Моды в этом инстансе не найдены",

	"resourcepacks":        "Показать ресурс-паки в инстансе",
	"resourcepacks.arg.id": "Инстанс для показа ресурс-паков",
	"resourcepacks.empty":  "Ресурс-паки в этом инстансе не найдены",

	"shaders":        "Показать шейдер-паки в инстансе",
	"shaders.arg.id": "Инстанс для показа шейдер-паков",
	"shaders.empty":  "Шейдер-паки в этом инстансе не найдены",

	"mods.table.name":       "Имя",
	"mods.table.curseforge": "CurseForge",
	"mods.table.modrinth":   "Modrinth",
	"mods.table.size":       "Размер",

	"resourcepacks.table.name":       "Имя",
	"resourcepacks.table.curseforge": "CurseForge",
	"resourcepacks.table.modrinth":   "Modrinth",
	"resourcepacks.table.size":       "Размер",
	"shaders.table.name":             "Имя",
	"shaders.table.size":             "Размер",

	"rename":          "Переименовать инстанс",
	"rename.complete": "Инстанс переименован.",
	"rename.arg.id":   "Инстанс для переименования",
	"rename.arg.new":  "Новое имя для инстанса",

	"search":               "Поиск версий",
	"search.complete":      "Найдено %d результатов",
	"search.table.version": "Версия",
	"search.table.type":    "Тип",
	"search.table.date":    "Дата релиза",
	"search.table.name":    "Имя",
	"instance.table.path":  "Путь",
	"java.table.name":      "Имя",
	"java.table.path":      "Путь",
	"search.arg.query":     "Поисковый запрос",
	"search.arg.kind":      "Что искать",
	"search.arg.reverse":   "Обратный порядок списка",

	"start":                    "Запустить инстанс",
	"start.arg.id":             "Инстанс для запуска",
	"start.arg.username":       "Имя пользователя (оффлайн режим)",
	"start.arg.server":         "Присоединиться к серверу при запуске игры",
	"start.arg.world":          "Присоединиться к миру при запуске игры",
	"start.arg.demo":           "Запустить игру в демо режиме",
	"start.arg.disablemp":      "Отключить мультиплеер",
	"start.arg.disablechat":    "Отключить чат",
	"start.arg.width":          "Ширина окна игры",
	"start.arg.height":         "Высота окна игры",
	"start.arg.jvm":            "Путь к JVM",
	"start.arg.jvmargs":        "Дополнительные аргументы JVM",
	"start.arg.minmemory":      "Минимальная память",
	"start.arg.maxmemory":      "Максимальная память",
	"start.arg.nojavawindow":   "Использовать javaw.exe вместо java.exe на Windows (без консольного окна)",
	"start.arg.prepare":        "Загрузить все необходимые ресурсы, но не запускать игру.",
	"start.arg.opts":           "Настройки игры",
	"start.arg.overrides":      "Переопределения конфигурации",
	"start.prepared":           "Игра успешно подготовлена.",
	"start.processing":         "Выполняются пост-обработки. Это может занять время.",
	"start.launch.downloading": "Загрузка файлов...",
	"start.launch.assets":      "Найдено %d ресурсов",
	"start.launch.libraries":   "Найдено %d библиотек",
	"start.launch.metadata":    "Метаданные версии загружены",
	"start.launch.jvmargs":     "Аргументы JVM: %s",
	"start.launch.gameargs":    "Аргументы игры: %s",
	"start.launch.info":        "Запуск главного класса %q. Директория игры: %q.",
	"start.launch":             "Запуск игры как %s...",

	"arg.verbosity": "Изменить уровень подробности вывода",
	"arg.dir":       "Корневая директория для файлов лаунчера",
	"arg.nocolor":   "Отключить всю цветовую подсветку. Также поддерживается переменная окружения NO_COLOR.",

	"update.current_version": "Текущая версия",
	"update.platform":        "Платформа",

	"tip.internet":  "Проверьте подключение к интернету.",
	"tip.cache":     "Удаленные ресурсы не были кэшированы и не могут быть получены. Проверьте подключение к интернету.",
	"tip.configure": "Настройте этот инстанс с помощью файла `instance.toml` в директории инстанса.",
	"tip.nojvm":     "Если JVM от Mojang недоступно, вы можете установить его самостоятельно и указать путь к исполняемому файлу Java в конфигурации инстанса.",
	"tip.noaccount": "Для запуска в оффлайн режиме используйте флаг --username (-u).",

	"launcher.description": "Минималистичный лаунчер Minecraft для командной строки.",
	"launcher.license":     "Лицензия MIT",
	"launcher.copyright":   "Copyright 2024-2025 telecter",
	"launcher.error":       "Ошибка",
	"launcher.warning":     "Предупреждение",
	"launcher.debug":       "Отладка",
	"launcher.tip":         "Совет",
	"arg.lang":             "Язык вывода (ru, en)",
	"cli.title":            "QMLauncher CLI",
	"cli.subtitle":         "Лаунчер Minecraft с поддержкой модов",
	"cli.usage":            "ИСПОЛЬЗОВАНИЕ:",
	"cli.usage.cmd":        "  QMLauncher [команда]",
	"cli.commands":         "ДОСТУПНЫЕ КОМАНДЫ:",
	"cli.cmd.start":        "  start       Запустить Minecraft с указанными опциями",
	"cli.cmd.instance":     "  instance, -i         Управление инстансами Minecraft",
	"cli.cmd.update":       "  update               Обновить лаунчер",
	"cli.cmd.auth":         "  auth                 Управление аутентификацией",
	"cli.cmd.search":       "  search               Поиск версий Minecraft",
	"cli.cmd.java":         "  java                 Управление установками Java",
	"cli.cmd.about":        "  about                Показать информацию о версии",
	"cli.aliases":          "АЛИАСЫ:",
	"cli.alias.i":          "  -i                   Сокращение для команды 'instance'",
	"cli.alias.s":          "  -s                   Сокращение для команды 'start' (в контексте instance)",
	"cli.alias.is":         "  -is                  Сокращение для 'instance start'",
	"cli.help":             "Используйте 'QMLauncher [команда] --help' для получения дополнительной информации о команде.",
}

var lang = ru

// SetLang changes the language to the specified language, if translations for it exist.
func SetLang(tag language.Tag) {
	switch tag {
	case language.Russian:
		lang = ru
	case language.English:
		lang = en
	default:
		// Default to Russian for all other languages
		lang = ru
	}
}

// Translations returns the map of output translations for the current language.
func Translations() map[string]string {
	return lang
}

// Translate takes a translation string and looks up its human-readable text. If not available, it returns the same translation string.
func Translate(key string) string {
	t, ok := lang[key]
	if !ok {
		return key
	}
	return t
}
