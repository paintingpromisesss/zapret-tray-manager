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
	VPNManage             string
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

	UpdateUpToDateHeading string

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
	VPNManage:             "Pause zapret while VPN is connected",
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

	UpdateUpToDateHeading: "You are running the latest version (v%s)",

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
	VPNManage:             "Приостанавливать zapret при подключении VPN",
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

	UpdateUpToDateHeading: "Установлена последняя версия (v%s)",

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
