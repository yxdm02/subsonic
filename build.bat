@echo off

echo [*] Navigating to 'subsonic' directory...
cd subsonic

REM 1. Build the frontend
echo [*] Building frontend...
cd frontend
call npm install
IF %ERRORLEVEL% NEQ 0 (
    echo [!] npm install failed.
    exit /b %ERRORLEVEL%
)
call npm run build
IF %ERRORLEVEL% NEQ 0 (
    echo [!] Frontend build failed.
    exit /b %ERRORLEVEL%
)
cd ..

REM 2. Build the Go backend
echo [*] Building backend...
go build -ldflags "-w -s" -o subsonic.exe main.go
IF %ERRORLEVEL% NEQ 0 (
    echo [!] Backend build failed.
    exit /b %ERRORLEVEL%
)

echo [+] Build complete! Run subsonic.exe from the 'subsonic' directory.
