@echo off
setlocal
cd /d "%~dp0"
echo === helmme-ai dev: pg(:5433) api(:8081) web(:3001) ===

REM --- Postgres (docker, host :5433; 5432 belongs to another project) ---
REM Supports both names: helmme-pg (standalone) and helmme-ai-postgres-1 (compose).
docker exec helmme-ai-postgres-1 pg_isready -U helmme >nul 2>&1
if errorlevel 1 docker exec helmme-pg pg_isready -U helmme >nul 2>&1
if errorlevel 1 (
  echo [pg] starting helmme-pg...
  docker rm -f helmme-pg >nul 2>&1
  docker run -d --name helmme-pg -e POSTGRES_USER=helmme -e POSTGRES_PASSWORD=helmme -e POSTGRES_DB=helmme -p 5433:5432 postgres:16-alpine >nul
  timeout /t 10 /nobreak >nul
  docker exec -i helmme-pg psql -U helmme -d helmme < db\migrations\001_init.sql
) else (
  echo [pg] up
)

REM --- API (:8081; 8080 belongs to another project) ---
powershell -NoProfile -Command "try { (Invoke-WebRequest http://localhost:8081/healthz -UseBasicParsing -TimeoutSec 3) | Out-Null; exit 0 } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  echo [api] building + starting on :8081...
  if not exist "%TEMP%\helmme-api.exe" (
    pushd services\api
    go build -o "%TEMP%\helmme-api.exe" .
    popd
  )
  start "helmme-api" /min powershell -NoProfile -Command "$env:DATABASE_URL='postgres://helmme:helmme@localhost:5433/helmme?sslmode=disable'; $env:PORT='8081'; & '%TEMP%\helmme-api.exe'"
  timeout /t 5 /nobreak >nul
) else (
  echo [api] up
)

REM --- Web omnibar (:3001; 3000 belongs to another project) ---
powershell -NoProfile -Command "try { (Invoke-WebRequest http://localhost:3001 -UseBasicParsing -TimeoutSec 3) | Out-Null; exit 0 } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  echo [web] starting on :3001 - first boot compiles ~30s...
  pushd "%~dp0apps\web"
  start "helmme-web" /min npm run dev -- --port 3001
  popd
) else (
  echo [web] up
)

echo.
echo   API : http://localhost:8081/healthz
echo   Web : http://localhost:3001
echo.
pause
