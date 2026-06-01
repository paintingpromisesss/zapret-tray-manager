package i18n

const AppTitle = "Zapret Tray Manager"

type Strings struct {
	ServiceUnknown  string
	StrategyUnknown string
	ErrorNone       string
	ServicePrefix   string
	StrategyPrefix  string
	StrategyNone    string
	ErrorPrefix     string
	RootNotSet      string

	Run  string
	Stop string

	Strategies        string
	AddCustomStrategy string
	UpdateIPSet       string

	Settings              string
	AutoRunService        string
	AutostartWithWindows  string
	GlobalIPSetGameFilter string
	VPNInteraction        string
	VPNStopOnConnect      string
	VPNStartOnDisconnect  string
	ZapretVersions        string
	LocalZapret           string
	AddLocalZapret        string
	Logs                  string
	ViewLogs              string
	ExportLogs            string
	OpenProgramFolder     string
	ShowServiceInfo       string
	RemoveServices        string
	RefreshVersions       string
	CheckForUpdates       string
	UpdateAvailableMenu   string

	Quit                 string
	WorkingBusy          string
	ZapretFolderNotFound string
	OpenPage             string

	UpdateAvailableTitle   string
	UpdateAvailableHeading string
	UpdateAvailableBody    string
	UpdateUpToDateTitle    string
	UpdateUpToDateHeading  string
	UpdateUpToDateBody     string

	LanguageMenu string
	LangEnglish  string
	LangRussian  string
}

var en = Strings{
	ServiceUnknown:  "Service: unknown",
	StrategyUnknown: "Strategy: unknown",
	ErrorNone:       "Error: none",
	ServicePrefix:   "Service: ",
	StrategyPrefix:  "Strategy: ",
	StrategyNone:    "none",
	ErrorPrefix:     "Error: ",
	RootNotSet:      "(not set)",

	Run:  "Run",
	Stop: "Stop",

	Strategies:        "Strategies",
	AddCustomStrategy: "Add custom strategy...",
	UpdateIPSet:       "Update IPSet list",

	Settings:              "Settings",
	AutoRunService:        "Auto-run service",
	AutostartWithWindows:  "Autostart with Windows",
	GlobalIPSetGameFilter: "Global IPSet/GameFilter",
	VPNInteraction:        "VPN interaction",
	VPNStopOnConnect:      "Stop when VPN connects",
	VPNStartOnDisconnect:  "Start when VPN disconnects",
	ZapretVersions:        "Zapret versions",
	LocalZapret:           "Local zapret",
	AddLocalZapret:        "Add local zapret...",
	Logs:                  "Logs",
	ViewLogs:              "View logs",
	ExportLogs:            "Export logs...",
	OpenProgramFolder:     "Open program folder",
	ShowServiceInfo:       "Show service info",
	RemoveServices:        "Remove services",
	RefreshVersions:       "Refresh versions",
	CheckForUpdates:       "Check for updates",
	UpdateAvailableMenu:   "Update available (%s)",

	Quit:                 "Quit",
	WorkingBusy:          "Working: busy",
	ZapretFolderNotFound: "Zapret folder not found",
	OpenPage:             "(open page)",

	UpdateAvailableTitle:   "Update available",
	UpdateAvailableHeading: "A new version %s is available",
	UpdateAvailableBody:    "Current version: %s.\n\nDownload and install now?",
	UpdateUpToDateTitle:    "No updates",
	UpdateUpToDateHeading:  "You are up to date",
	UpdateUpToDateBody:     "You are running the latest version (%s).",

	LanguageMenu: "Language",
	LangEnglish:  "English",
	LangRussian:  "Русский",
}

var ru = Strings{
	ServiceUnknown:  "Служба: неизвестно",
	StrategyUnknown: "Стратегия: неизвестно",
	ErrorNone:       "Ошибка: нет",
	ServicePrefix:   "Служба: ",
	StrategyPrefix:  "Стратегия: ",
	StrategyNone:    "нет",
	ErrorPrefix:     "Ошибка: ",
	RootNotSet:      "(не задано)",

	Run:  "Запустить",
	Stop: "Остановить",

	Strategies:        "Стратегии",
	AddCustomStrategy: "Добавить стратегию...",
	UpdateIPSet:       "Обновить IPSet",

	Settings:              "Настройки",
	AutoRunService:        "Автозапуск Zapret",
	AutostartWithWindows:  "Автозапуск программы",
	GlobalIPSetGameFilter: "Глобальные IPSet/GameFilter",
	VPNInteraction:        "Взаимодействие с VPN",
	VPNStopOnConnect:      "Остановить при подключении VPN",
	VPNStartOnDisconnect:  "Запустить при отключении VPN",
	ZapretVersions:        "Версии Zapret",
	LocalZapret:           "Локальный zapret",
	AddLocalZapret:        "Добавить локальный zapret...",
	Logs:                  "Логи",
	ViewLogs:              "Просмотр логов",
	ExportLogs:            "Экспорт логов...",
	OpenProgramFolder:     "Открыть папку программы",
	ShowServiceInfo:       "Информация о службах",
	RemoveServices:        "Удалить службы",
	RefreshVersions:       "Обновить список версий",
	CheckForUpdates:       "Проверить обновления",
	UpdateAvailableMenu:   "Доступно обновление (%s)",

	Quit:                 "Выход",
	WorkingBusy:          "Выполняется...",
	ZapretFolderNotFound: "Папка zapret не найдена",
	OpenPage:             "(открыть страницу)",

	UpdateAvailableTitle:   "Доступно обновление",
	UpdateAvailableHeading: "Доступна новая версия %s",
	UpdateAvailableBody:    "Текущая версия: %s.\n\nСкачать и установить сейчас?",
	UpdateUpToDateTitle:    "Обновлений нет",
	UpdateUpToDateHeading:  "Установлена последняя версия",
	UpdateUpToDateBody:     "Установлена последняя версия (%s).",

	LanguageMenu: "Язык",
	LangEnglish:  "English",
	LangRussian:  "Русский",
}

func Load(lang string) *Strings {
	if lang == "russian" || lang == "ru" {
		return &ru
	}
	return &en
}
