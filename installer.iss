#define AppName "Zapret Tray Manager"
#ifndef AppVersion
  #define AppVersion "1.0.0"
#endif
#define AppExeName "Zapret Tray Manager.exe"
#define AppPublisher "paintingpromisesss"

[Setup]
AppId={{A3F7B2C1-4D8E-4F9A-B123-56789ABCDEF0}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
OutputDir=dist
OutputBaseFilename={#AppName}-{#AppVersion}-setup
Compression=lzma2
SolidCompression=yes
PrivilegesRequired=admin
SetupIconFile=assets\rkn_blocked_icon.ico
UninstallDisplayIcon={app}\{#AppExeName}
WizardStyle=modern

[Languages]
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "build\{#AppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[Dirs]
Name: "{app}\custom_strategies"

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"
Name: "{group}\Uninstall {#AppName}"; Filename: "{uninstallexe}"

[Run]
Filename: "schtasks.exe"; \
  Parameters: "/Create /TN ""ZapretTrayManager"" /TR ""{app}\{#AppExeName}"" /SC ONLOGON /RL HIGHEST /F"; \
  Flags: runhidden
Filename: "{app}\{#AppExeName}"; \
  Parameters: "--lang={language}"; \
  Flags: nowait postinstall skipifsilent runascurrentuser; \
  Description: "Запустить {#AppName}"

[Code]
function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  Exec('taskkill.exe', '/f /im "{#AppExeName}"', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := '';
end;

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/f /im {#AppExeName}"; Flags: runhidden; RunOnceId: "KillApp"
Filename: "schtasks.exe"; Parameters: "/Delete /TN ""ZapretTrayManager"" /F"; Flags: runhidden; RunOnceId: "DeleteTask"
