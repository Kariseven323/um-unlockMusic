@echo off
chcp 65001 >nul
echo 刷新Windows图标缓存
echo =====================

echo 正在清除图标缓存...

:: 方法1：使用ie4uinit清除缓存
echo 使用ie4uinit清除缓存...
ie4uinit.exe -show >nul 2>&1
ie4uinit.exe -ClearIconCache >nul 2>&1

:: 方法2：删除图标缓存文件
echo 删除图标缓存文件...
del /f /q "%localappdata%\IconCache.db" >nul 2>&1
del /f /q "%localappdata%\Microsoft\Windows\Explorer\iconcache*" >nul 2>&1

:: 方法3：重启资源管理器
echo 重启Windows资源管理器...
taskkill /f /im explorer.exe >nul 2>&1
timeout /t 2 /nobreak >nul
start explorer.exe

echo.
echo 图标缓存已刷新！
echo 如果图标仍未更新，请重启计算机。
echo.
pause
