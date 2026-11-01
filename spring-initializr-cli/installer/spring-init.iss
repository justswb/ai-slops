#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

[Setup]
AppId={{8E79475C-9192-45BA-AEB4-B2BB84056A3B}
AppName=spring-init
AppVersion={#AppVersion}
DefaultDirName={localappdata}\Programs\spring-init
DisableDirPage=yes
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ChangesEnvironment=yes
OutputDir=..\dist
OutputBaseFilename=spring-init-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
UninstallDisplayIcon={app}\spring-init.exe

[Files]
Source: "..\dist\spring-init.exe"; DestDir: "{app}"; Flags: ignoreversion

[Code]
function NormalizePath(Value: String): String;
begin
  Value := Trim(Value);
  if (Length(Value) >= 2) and (Value[1] = '"') and
     (Value[Length(Value)] = '"') then
    Value := Copy(Value, 2, Length(Value) - 2);
  while (Length(Value) > 0) and (Value[Length(Value)] = '\') do
    Delete(Value, Length(Value), 1);
  Result := Lowercase(Value);
end;

{ Preserve the other entries, including unexpanded %VARIABLES% and empty entries. }
procedure UpdatePath(Add: Boolean);
var
  OldPath, NewPath, Entry, AppPath: String;
  Start, I: Integer;
  Found, HasEntry: Boolean;
begin
  if not RegQueryStringValue(HKCU, 'Environment', 'Path', OldPath) then
  begin
    if RegValueExists(HKCU, 'Environment', 'Path') then
      RaiseException('Could not read the user PATH.');
    OldPath := '';
  end;
  AppPath := ExpandConstant('{app}');
  NewPath := '';
  Found := False;
  HasEntry := False;
  Start := 1;
  for I := 1 to Length(OldPath) + 1 do
  begin
    if (I > Length(OldPath)) or (OldPath[I] = ';') then
    begin
      Entry := Copy(OldPath, Start, I - Start);
      if NormalizePath(Entry) = NormalizePath(AppPath) then
      begin
        if Add and not Found then
        begin
          if HasEntry then NewPath := NewPath + ';';
          NewPath := NewPath + Entry;
          HasEntry := True;
        end;
        Found := True;
      end
      else
      begin
        if HasEntry then NewPath := NewPath + ';';
        NewPath := NewPath + Entry;
        HasEntry := True;
      end;
      Start := I + 1;
    end;
  end;
  if Add and not Found then
  begin
    if NewPath <> '' then NewPath := NewPath + ';';
    NewPath := NewPath + AppPath;
  end;
  if NewPath <> OldPath then
    if not RegWriteExpandStringValue(HKCU, 'Environment', 'Path', NewPath) then
      RaiseException('Could not update the user PATH.');
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then UpdatePath(True);
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then UpdatePath(False);
end;
