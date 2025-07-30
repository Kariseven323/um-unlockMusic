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

# 确保um.exe存在
if not os.path.exists(um_exe_path):
    print(f"错误: 找不到um.exe文件: {um_exe_path}")
    sys.exit(1)

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
    icon=None,  # 可以在这里添加图标文件路径
    version_file=None,  # 可以在这里添加版本信息文件
)
