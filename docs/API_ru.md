# API QMLauncher (1.0.0)

В дополнение к CLI, этот лаунчер также предоставляет API, который можно использовать для программного взаимодействия с ним.
Эта страница будет структурирована аналогично основному README, но для API, а не CLI.

### Создание инстанса

Сначала вам нужно создать инстанс для запуска игры. Например:

```go
inst, err := launcher.CreateInstance(launcher.InstanceOptions{
		GameVersion: "1.21.5",
		Name:        "MyInstance",
		Loader:      launcher.LoaderQuilt,
		LoaderVersion: "latest",
		Config: launcher.InstanceConfig{}
})
```

Поле `Loader` может быть `LoaderQuilt`, `LoaderFabric` или `LoaderVanilla`.

`LoaderVersion`, если загрузчик модов не vanilla, может быть установлено в `latest` для последней версии или в конкретную версию этого загрузчика модов.

Поле `Config` - это структура InstanceConfig с опциями игры. Обратитесь к справочнику Go, чтобы найти её поля.

### Получение инстанса

Если у вас уже есть созданный инстанс, вы можете использовать функцию `FetchInstance` для его получения.

```go
inst, err := launcher.FetchInstance("MyInstance")
```

Инстансы можно удалять с помощью функции `launcher.RemoveInstance`, которая следует той же структуре.

Их также можно переименовывать с помощью метода `.Rename`.

Если вы хотите изменить конфигурацию инстанса, измените его поле `Config`, а затем запустите метод инстанса `WriteConfig`.

### Подготовка игры

После создания инстанса, чтобы запустить игру, вам нужно подготовить среду запуска.

```go
env, err := launcher.Prepare(inst, launcher.LaunchOptions{
	Session: auth.Session{
		Username: "Dinnerbone",
	},
	InstanceConfig: inst.Config,
}, myWatcher)
```

В LaunchOptions есть больше полей, которые можно найти в справочнике Go. Они позволяют настраивать поведение запуска игры, такое как быстрый запуск мультиплеера/одиночной игры. Также они позволяют переопределять конфигурацию инстанса через поле `InstanceConfig`. Если вы не хотите этого делать, предоставьте поле `Config` инстанса там.

А что насчет `myWatcher`? Watcher используется для ответа на различные события, которые происходят во время подготовки инстанса. Вам нужно определить функцию EventWatcher.

```go
func myWatcher(event any) {
	switch e := event.(type) {
	case launcher.DownloadingEvent:
		...
    case launcher.LibrariesResolvedEvent:
        ...
    case launcher.AssetsResolvedEvent:
        ...
	}
}
```

Это может быть использовано, например, для создания прогресс-бара загрузки библиотек/ассетов.

Поле `Session` установлено без access token, что означает, что это оффлайн-сессия. Мы перейдем к аутентификации позже.

### Запуск игры

После получения среды запуска из `launcher.Prepare` вам нужно создать `Runner`, который является просто другой функцией, чтобы фактически запустить и мониторить игру так, как вы хотите.

```go
func myRunner(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

Этот runner принимает `*exec.Cmd` и запускает его, копируя его потоки ввода-вывода в консоль. Этот случай использования, вероятно, будет довольно распространенным, поэтому есть реализация `launcher.ConsoleRunner`, которая делает именно это.
Однако вы должны использовать свой собственный runner, если у вас другой случай использования, такой как копирование логов в JSON и т.д.

С этим runner'ом вы можете запустить игру.

```go
err := launcher.Launch(env, myRunner)
```

### Аутентификация

Чтобы аутентифицироваться, вам нужно иметь Microsoft Azure приложение. После его создания скопируйте Client ID для использования здесь. Вы, вероятно, также захотите выбрать localhost redirect URI в дашборде Azure. **Убедитесь, что включили порт для использования!**

Инициализируйте значения следующим образом:

```go
// Вы захотите поместить это в функцию init() или sometime перед запуском любых функций аутентификации
func init() {
	auth.ClientID = "your client ID"
	// нужно, если вы хотите использовать auth code flow, который требует redirect
	auth.RedirectURI = "your redirect URI"
}
```

**OAuth2 auth code flow**
Сначала вам нужно получить auth URL для перехода пользователя.

```go
url := auth.AuthCodeURL()
```

Затем вам нужно либо отобразить эту ссылку пользователю, либо открыть окно браузера для аутентификации пользователя. После этого дождитесь, пока пользователь аутентифицируется с redirect:

```go
session, err := auth.AuthenticateWithRedirect()
```

И тогда у вас будет ваша сессия!

**OAuth2 device code flow**
Сначала получите device code и auth link для перехода пользователя.

```go
resp, err := auth.FetchDeviceCode()
```

Возвращается `deviceCodeResponse`. `UserCode` и `VerificationURI` должны быть отображены пользователю, чтобы он мог аутентифицироваться.
Затем вы должны опрашивать endpoint для обновлений аутентификации, чтобы получить вашу сессию:

```go
session, err := auth.AuthenticateWithCode(resp)
```

**Обновление**
Вы определенно не захотите использовать ссылку или auth code каждый раз, поэтому если у вас уже есть MSA refresh token, вы можете просто обновить данные аутентификации. Для этого используйте функцию `Authenticate`:

```go
session, err := auth.Authenticate()
```

Это обновит любые истекшие токены и данные и даст вам сессию.
И это всё! Вы можете использовать эту сессию в функции `launcher.Prepare`. Данные аутентификации автоматически сохраняются в файл env.AuthStorePath.
