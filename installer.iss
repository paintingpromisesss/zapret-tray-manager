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
SetupIconFile=internal\assets\rkn_blocked_icon.ico
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

procedure CurStepChanged(CurStep: TSetupStep);
var
  ExePath, Params: String;
  ResultCode: Integer;
begin
  if CurStep = ssPostInstall then
  begin
    ExePath := ExpandConstant('{app}\{#AppExeName}');
    Params := '/Create /TN "ZapretTrayManager" /TR "\"' + ExePath + '\"" /SC ONLOGON /RL HIGHEST /F';
    Exec('schtasks.exe', Params, '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  end;
end;

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/f /im {#AppExeName}"; Flags: runhidden; RunOnceId: "KillApp"
Filename: "schtasks.exe"; Parameters: "/Delete /TN ""ZapretTrayManager"" /F"; Flags: runhidden; RunOnceId: "DeleteTask"
