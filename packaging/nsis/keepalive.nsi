!include "MUI2.nsh"
!include "WordFunc.nsh"

!ifndef VERSION
  !define VERSION "0.0.0"
!endif

Name "Keepalive ${VERSION}"
OutFile "keepalive_${VERSION}_windows_amd64_setup.exe"
InstallDir "$PROGRAMFILES\Keepalive"
InstallDirRegKey HKLM "Software\Keepalive" "InstallDir"
RequestExecutionLevel admin

!define MUI_ICON "..\..\assets\icon.ico"
!define MUI_UNICON "..\..\assets\icon.ico"
!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File "keepalive.exe"

  ; Add to PATH
  ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
  StrCpy $0 "$0;$INSTDIR"
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path" "$0"
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  ; Start Menu shortcuts
  CreateDirectory "$SMPROGRAMS\Keepalive"
  CreateShortCut "$SMPROGRAMS\Keepalive\Uninstall Keepalive.lnk" "$INSTDIR\uninstall.exe"

  ; Uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Add/Remove Programs registry
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive" "DisplayName" "Keepalive"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive" "DisplayIcon" "$INSTDIR\keepalive.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive" "Publisher" "jecortes2304"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive" "InstallLocation" "$INSTDIR"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive" "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive" "NoRepair" 1

  ; Store install dir
  WriteRegStr HKLM "Software\Keepalive" "InstallDir" "$INSTDIR"
SectionEnd

Section "Uninstall"
  ; Remove files
  Delete "$INSTDIR\keepalive.exe"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  ; Remove from PATH
  ReadRegStr $0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
  ${WordReplace} "$0" ";$INSTDIR" "" "+" $0
  WriteRegExpandStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path" "$0"
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  ; Remove Start Menu
  Delete "$SMPROGRAMS\Keepalive\Uninstall Keepalive.lnk"
  RMDir "$SMPROGRAMS\Keepalive"

  ; Remove registry
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Keepalive"
  DeleteRegKey HKLM "Software\Keepalive"
SectionEnd
