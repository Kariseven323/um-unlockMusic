# -*- mode: python ; coding: utf-8 -*-

import os
import sys

# 获取项目根目录 - build.spec在music_unlock_gui/build_config/目录下
# 所以需要向上两级到达项目根目录
spec_dir = os.path.dirname(os.path.abspath(SPEC))  # build_config目录
gui_dir = os.path.dirname(spec_dir)  # music_unlock_gui目录
project_root = os.path.dirname(gui_dir)  # 项目根目录

# um.exe的路径
um_exe_path = os.path.join(project_root, 'um.exe')

# 图标文件路径
icon_path = os.path.join(gui_dir, 'resources', '音乐解密软件图标设计.png')

# 确保um.exe存在
if not os.path.exists(um_exe_path):
    print(f"错误: 找不到um.exe文件: {um_exe_path}")
    sys.exit(1)

# 检查图标文件是否存在
if not os.path.exists(icon_path):
    print(f"警告: 找不到图标文件: {icon_path}")
    icon_path = None
else:
    print(f"使用图标文件: {icon_path}")
    # 检查文件大小，确保不是空文件
    file_size = os.path.getsize(icon_path)
    print(f"图标文件大小: {file_size} 字节")
    if file_size == 0:
        print("警告: 图标文件为空")
        icon_path = None

block_cipher = None

a = Analysis(
    [os.path.join(gui_dir, 'main.py')],
    pathex=[project_root],
    binaries=[],
    datas=[
        # 将um.exe打包到程序中
        (um_exe_path, '.'),
    ],
    hiddenimports=[],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name='MusicUnlockGUI',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,  # 不显示控制台窗口
    disable_windowed_traceback=False,
    target_arch=None,
    codesign_identity=None,
    entitlements_file=None,
    icon=icon_path,  # 使用PNG图标文件（PyInstaller会自动转换）
    version_file=None,  # 可以在这里添加版本信息文件
)
