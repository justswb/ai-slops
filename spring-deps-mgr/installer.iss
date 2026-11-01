#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

[Setup]
AppId=justswb.spring-deps-mgr
AppName=spring-deps-mgr
AppVersion={#AppVersion}
AppPublisher=justswb
AppPublisherURL=https://github.com/justswb/ai-slops
DefaultDirName={localappdata}\Programs\spring-deps-mgr
DisableDirPage=yes
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64
OutputDir=dist
OutputBaseFilename=spring-deps-mgr-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ChangesEnvironment=yes
UninstallDisplayIcon={app}\spring-deps-mgr.exe
CloseApplications=yes
RestartApplications=no
LicenseFile=..\LICENSE

[Files]
Source: "dist\spring-deps-mgr.exe"; DestDir: "{app}"; Flags: ignoreversion

[Messages]
FinishedLabel=Setup has installed spring-deps-mgr for your user account and added it to your user PATH.

[Code]
function NormalizePath(Value: String): String;
begin
  Value := Trim(Value);
  if (Length(Value) > 1) and (Value[1] = '"') and (Value[Length(Value)] = '"') then
    Value := Copy(Value, 2, Length(Value) - 2);
  Result := Lowercase(RemoveBackslashUnlessRoot(Value));
end;

{ Remove only our entry, preserving other entries, empty entries, order and %VARS%. }
function WithoutDirectory(Value, Directory: String; var Found: Boolean): String;
var
  Entry: String;
  Separator: Integer;
  HaveEntry: Boolean;
begin
  Result := '';
  Found := False;
  HaveEntry := False;
  while True do begin
    Separator := Pos(';', Value);
    if Separator = 0 then Entry := Value
    else Entry := Copy(Value, 1, Separator - 1);
    if NormalizePath(Entry) = NormalizePath(Directory) then Found := True
    else begin
      if HaveEntry then Result := Result + ';';
      Result := Result + Entry;
      HaveEntry := True;
    end;
    if Separator = 0 then Break;
    Delete(Value, 1, Separator);
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  Current, Unused, Directory: String;
  Found: Boolean;
begin
  if CurStep <> ssPostInstall then Exit;
  Directory := ExpandConstant('{app}');
  RegQueryStringValue(HKCU, 'Environment', 'Path', Current);
  Unused := WithoutDirectory(Current, Directory, Found);
  if not Found then begin
    if Current <> '' then Current := Current + ';';
    if not RegWriteExpandStringValue(HKCU, 'Environment', 'Path', Current + Directory) then
      RaiseException('Could not add the installation directory to the user PATH.');
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Current, Updated: String;
  Found: Boolean;
begin
  if CurUninstallStep <> usPostUninstall then Exit;
  RegQueryStringValue(HKCU, 'Environment', 'Path', Current);
  Updated := WithoutDirectory(Current, ExpandConstant('{app}'), Found);
  if Found and not RegWriteExpandStringValue(HKCU, 'Environment', 'Path', Updated) then
    RaiseException('Could not remove the installation directory from the user PATH.');
end;
