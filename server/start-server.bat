@echo off
chcp 65001 >nul
cd /d "%~dp0"
title 智能刷题助手 API 后端服务 (端口 8080)

echo ===================================================
echo     正在启动 智能刷题助手 API 后端服务...
echo ===================================================
echo 服务端口: 8080
echo 探活地址: http://127.0.0.1:8080/health
echo.

netstat -ano | findstr :8080 | findstr LISTENING >nul 2>nul
if %ERRORLEVEL% equ 0 (
    echo [提示] 端口 8080 已被占用，可能有后端服务正在运行中。
    echo 如需重新启动，请先关闭已有的服务窗口或终止端口进程。
    echo ---------------------------------------------------
)

if exist "api-server.exe" goto RUN_EXE
goto RUN_SRC

:RUN_EXE
echo [启动中] 正在运行预编译的 api-server.exe...
api-server.exe
goto AFTER_RUN

:RUN_SRC
echo [启动中] 正在通过 Go 源码运行: go run ./cmd/api...
go run ./cmd/api
goto AFTER_RUN

:AFTER_RUN
if %ERRORLEVEL% neq 0 (
    echo.
    echo [提示] 服务已退出，退出代码: %ERRORLEVEL%
)

pause
