; Inno Setup script for the Windows installer (see docs/releasing.md).
;
;   iscc /DAppVersion=0.10.0 /DExe=..\bin\dist\OpenWA-windows-amd64.exe installer\wazzap.iss

#ifndef AppVersion
#define AppVersion "0.0.0"
#endif
#ifndef Exe
#define Exe "..\bin\dist\OpenWA-windows-amd64.exe"
#endif
#ifndef OutDir
#define OutDir "..\bin\dist"
#endif

[Setup]
AppId={{6B0F3B52-9C1E-4D7A-A3E5-2F7C8D41B9E6}
AppName=OpenWA
AppVersion={#AppVersion}
AppPublisher=Skidy89
AppPublisherURL=https://github.com/skidy89/openWA
AppSupportURL=https://github.com/skidy89/openWA/issues
AppUpdatesURL=https://github.com/skidy89/openWA/releases
DefaultDirName={autopf}\OpenWA
DefaultGroupName=OpenWA
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon={app}\OpenWA.exe
UninstallDisplayName=OpenWA
OutputDir={#OutDir}
OutputBaseFilename=OpenWA-Setup
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
SetupIconFile=wazzap.ico
CloseApplications=force
RestartApplications=no

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "{#Exe}"; DestDir: "{app}"; DestName: "OpenWA.exe"; Flags: ignoreversion
Source: "..\bin\dist\ffmpeg.exe"; DestDir: "{app}"; DestName: "ffmpeg.exe"; Flags: ignoreversion
Source: "..\bin\dist\ffmpeg-LICENSE.txt"; DestDir: "{app}"; DestName: "ffmpeg-LICENSE.txt"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\OpenWA"; Filename: "{app}\OpenWA.exe"; AppUserModelID: "OpenWA.Desktop"
Name: "{autodesktop}\OpenWA"; Filename: "{app}\OpenWA.exe"; AppUserModelID: "OpenWA.Desktop"; Tasks: desktopicon

[Run]
Filename: "{app}\OpenWA.exe"; Description: "{cm:LaunchProgram,OpenWA}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
Filename: "{sys}\taskkill.exe"; Parameters: "/f /im OpenWA.exe"; Flags: runhidden; RunOnceId: "QuitApp"

[UninstallDelete]
Type: files; Name: "{app}\OpenWA.exe.old"
Type: files; Name: "{app}\OpenWA.exe.new"

[Registry]
Root: HKCU; Subkey: "Software\Classes\AppUserModelId\OpenWA.Desktop"; ValueType: none; Flags: uninsdeletekey dontcreatekey
Root: HKCU; Subkey: "Software\Classes\CLSID{{EC56F81C-5D4A-4D8C-8044-761DDA20EDC8}"; ValueType: none; Flags: uninsdeletekey dontcreatekey
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueName: "OpenWA"; ValueType: none; Flags: uninsdeletevalue dontcreatekey

[Code]
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
Data: String;
begin
if CurUninstallStep <> usPostUninstall then
Exit;
Data := ExpandConstant('{userappdata}\WazzapClients');
if not DirExists(Data) or UninstallSilent then
Exit;
if MsgBox('Also delete your chats and log WazzapClients out of WhatsApp on this computer?' + #13#10#13#10 +
'Keep them to pick up where you left off if you install it again.',
mbConfirmation, MB_YESNO or MB_DEFBUTTON2) = IDYES then
DelTree(Data, True, True, True);
end;
