@echo off
chcp 65001 >nul
echo 音乐解密GUI工具 - 开发模式启动
echo ================================

:: 检查Python是否安装
python --version >nul 2>&1
if errorlevel 1 (
    echo 错误: 未找到Python，请先安装Python 3.7+
    pause
    exit /b 1
)

:: 检查um.exe是否存在
if not exist "um.exe" (
    echo 错误: 未找到um.exe文件，请确保um.exe在项目根目录
    pause
    exit /b 1
)

:: 启动程序
echo 启动音乐解密GUI工具...
python music_unlock_gui/run.py

pause
