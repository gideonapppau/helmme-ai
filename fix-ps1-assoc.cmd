@echo off
REM One-click repair: restores .ps1 double-click = run with PowerShell.
REM Needs ONE UAC "Yes" click. Safe to re-run.
echo === Repair .ps1 association (UAC prompt incoming - click Yes) ===
for /f "tokens=2 delims==" %%S in ('powershell -NoProfile -Command "([System.Security.Principal.WindowsIdentity]::GetCurrent()).User.Value"') do set SID=%%S
powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process powershell -Verb RunAs -Wait -ArgumentList '-NoProfile','-ExecutionPolicy','Bypass','-Command', 'assoc .ps1=Microsoft.PowerShellScript.1; ftype Microsoft.PowerShellScript.1=C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe -NoLogo -ExecutionPolicy Bypass -File %%1 %%*; Remove-Item Registry::HKEY_USERS\%SID%\Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts\.ps1\UserChoice -Recurse -Force; Write-Output REPAIRED; pause'"
echo Done. Double-click a .ps1 now runs it. (Old default was Notepad - by design.)
pause
