@echo off
chcp 65001 >nul
echo 音乐解密GUI工具 - 构建脚本
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

:: 安装PyInstaller
echo 正在安装PyInstaller...
pip install pyinstaller

:: 清理之前的构建
if exist "dist" rmdir /s /q "dist"
if exist "build" rmdir /s /q "build"

:: 开始构建
echo 开始构建可执行文件...
pyinstaller music_unlock_gui/build_config/build.spec

:: 检查构建结果
if exist "dist\MusicUnlockGUI.exe" (
    echo.
    echo ================================
    echo 构建成功！
    echo 可执行文件位置: dist\MusicUnlockGUI.exe
    echo ================================
    echo.
    
    :: 询问是否运行
    set /p choice="是否立即运行程序？(y/n): "
    if /i "%choice%"=="y" (
        start "" "dist\MusicUnlockGUI.exe"
    )
) else (
    echo.
    echo 构建失败，请检查错误信息
)

pause
