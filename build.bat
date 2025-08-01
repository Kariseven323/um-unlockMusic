@echo off
chcp 65001 >nul
echo 音乐解密GUI工具 - 构建脚本
echo ================================

:: 检查Go是否安装
echo 正在检查Go环境...
go version >nul 2>&1
if errorlevel 1 (
    echo 错误: 未找到Go，请先安装Go 1.23+
    echo 下载地址: https://golang.org/dl/
    pause
    exit /b 1
)

:: 检查Python是否安装
echo 正在检查Python环境...
python --version >nul 2>&1
if errorlevel 1 (
    echo 错误: 未找到Python，请先安装Python 3.7+
    pause
    exit /b 1
)

:: 构建Go后端
echo.
echo 正在构建Go后端 (um.exe)...
echo ================================

:: 清理旧的um.exe
if exist "um.exe" (
    echo 删除旧的um.exe...
    del "um.exe"
)

:: 编译Go程序
echo 正在编译Go程序...
go build -o um.exe ./cmd/um
if errorlevel 1 (
    echo 错误: Go程序编译失败
    pause
    exit /b 1
)

:: 验证um.exe生成成功
if not exist "um.exe" (
    echo 错误: um.exe生成失败
    pause
    exit /b 1
)

echo Go后端构建成功！

:: 构建Python GUI前端
echo.
echo 正在构建Python GUI前端...
echo ================================

:: 安装PyInstaller
echo 正在安装PyInstaller...
pip install pyinstaller

:: 清理之前的构建
echo 清理之前的构建文件...
if exist "dist" rmdir /s /q "dist"
if exist "build" rmdir /s /q "build"

:: 开始构建GUI
echo 正在构建GUI可执行文件...
pyinstaller music_unlock_gui/build_config/build.spec

:: 检查构建结果
echo.
echo 正在检查构建结果...
echo ================================

if exist "dist\MusicUnlockGUI.exe" (
    echo.
    echo ✓ 构建成功！
    echo ================================
    echo 后端程序: um.exe
    echo GUI程序: dist\MusicUnlockGUI.exe
    echo ================================
    echo.

    :: 询问是否运行
    set /p choice="是否立即运行GUI程序？(y/n): "
    if /i "%choice%"=="y" (
        start "" "dist\MusicUnlockGUI.exe"
    )
) else (
    echo.
    echo ✗ GUI构建失败，请检查错误信息
    echo 注意: Go后端 (um.exe) 已成功构建
)

pause
