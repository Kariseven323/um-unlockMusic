#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
常量定义模块
"""

# 默认支持的音频格式列表
DEFAULT_SUPPORTED_EXTENSIONS = [
    # 网易云音乐
    '.ncm',
    
    # QQ音乐
    '.qmc0', '.qmc2', '.qmc3', '.qmc4', '.qmc6', '.qmc8',
    '.qmcflac', '.qmcogg', '.tkm',
    '.mflac', '.mflac0', '.mflac1', '.mflaca', '.mflach', '.mflacl', '.mflacm',
    '.mgg', '.mgg0', '.mgg1', '.mgga', '.mggh', '.mggl', '.mggm',
    '.mmp4',
    
    # QQ音乐微云格式
    '.666c6163', '.6d3461', '.6d7033', '.6f6767', '.776176',
    
    # 酷狗音乐
    '.kgm', '.kgma', '.kgg', '.kgm.flac',
    
    # 酷我音乐
    '.kwm',
    
    # 太合音乐
    '.tm0', '.tm2', '.tm3', '.tm6',
    
    # 虾米音乐
    '.xm', '.x2m', '.x3m',
    
    # Moo Music
    '.bkcmp3', '.bkcm4a', '.bkcflac', '.bkcwav', '.bkcape', '.bkcogg', '.bkcwma',
    
    # 其他
    '.vpr', '.vpr.flac'
]

# 平台格式分类配置
PLATFORM_FORMAT_GROUPS = {
    "网易云音乐": {
        "keywords": [".ncm"],
        "description": "网易云音乐加密格式"
    },
    "QQ音乐": {
        "keywords": [".qmc", ".mflac", ".mgg", ".tkm", ".mmp4", ".666c6163", ".6d3461", ".6d7033", ".6f6767", ".776176"],
        "description": "QQ音乐及相关加密格式"
    },
    "酷狗音乐": {
        "keywords": [".kgm", ".kgg"],
        "description": "酷狗音乐加密格式"
    },
    "酷我音乐": {
        "keywords": [".kwm"],
        "description": "酷我音乐加密格式"
    },
    "太合音乐": {
        "keywords": [".tm"],
        "description": "太合音乐加密格式"
    },
    "虾米音乐": {
        "keywords": [".xm", ".x2m", ".x3m"],
        "description": "虾米音乐加密格式"
    },
    "Moo Music": {
        "keywords": [".bkc"],
        "description": "Moo Music加密格式"
    },
    "其他格式": {
        "keywords": [".vpr"],
        "description": "其他平台加密格式"
    }
}

# 解密后的输出格式
OUTPUT_FORMATS = {
    "音频格式": {
        "formats": [".mp3", ".flac", ".ogg", ".m4a", ".wav", ".aac"],
        "description": "解密后生成的音频文件格式"
    }
}

# 删除工具相关常量
DELETE_TOOL_MESSAGES = {
    'no_formats_selected': "请至少选择一种文件格式",
    'no_folder_selected': "请选择要处理的文件夹",
    'folder_not_exists': "所选文件夹不存在",
    'scan_started': "开始扫描文件...",
    'scan_completed': "扫描完成",
    'delete_started': "开始删除文件...",
    'delete_completed': "删除操作完成",
    'delete_cancelled': "删除操作已取消",
    'confirm_delete': "确认删除 {} 个文件？\n\n此操作不可撤销！",
    'no_files_found': "未找到符合条件的文件"
}

# 输出模式常量
OUTPUT_MODE_SOURCE = "source"
OUTPUT_MODE_CUSTOM = "custom"

# 命名格式常量
NAMING_FORMAT_AUTO = "auto"
NAMING_FORMAT_TITLE_ARTIST = "title-artist"
NAMING_FORMAT_ARTIST_TITLE = "artist-title"
NAMING_FORMAT_ORIGINAL = "original"

# 命名格式显示文本
NAMING_FORMAT_LABELS = {
    NAMING_FORMAT_AUTO: "智能识别（推荐）",
    NAMING_FORMAT_TITLE_ARTIST: "歌曲名-歌手名",
    NAMING_FORMAT_ARTIST_TITLE: "歌手名-歌曲名",
    NAMING_FORMAT_ORIGINAL: "保持原文件名"
}

# UI相关常量
UI_WINDOW_TITLE = "音乐解密工具 - Unlock Music GUI"
UI_WINDOW_SIZE = "900x700"
UI_WINDOW_MIN_SIZE = (800, 600)

# 删除工具窗口常量
DELETE_TOOL_WINDOW_TITLE = "文件删除工具"
DELETE_TOOL_WINDOW_SIZE = "800x600"
DELETE_TOOL_WINDOW_MIN_SIZE = (700, 500)

# 处理相关常量
DEFAULT_MAX_WORKERS = 6
PROCESS_TIMEOUT_SECONDS = 300  # 5分钟
UM_COMMAND_TIMEOUT = 10  # um.exe命令超时时间

# 日志相关常量
LOG_FORMAT = '%(asctime)s - %(name)s - %(levelname)s - %(message)s'

# 错误消息常量
ERROR_MESSAGES = {
    'file_not_found': "输入文件不存在: {}",
    'unsupported_format': "不支持的文件格式: {}",
    'no_files_selected': "请先添加要转换的文件",
    'no_output_dir': "请先选择输出目录",
    'processing_timeout': "处理超时（超过5分钟）",
    'processing_exception': "处理异常: {}",
    'um_exe_not_found': "um.exe not found at: {}",
    'conversion_failed': "转换失败: {}",
    'already_processing': "正在处理文件，无法清除列表"
}

# 成功消息常量
SUCCESS_MESSAGES = {
    'conversion_success': "成功转换: {}",
    'files_added': "已添加 {} 个文件，总计 {} 个文件",
    'list_cleared': "文件列表已清除",
    'processing_started': "开始转换...",
    'processing_stopped': "转换已停止"
}
