# 音乐解密GUI工具

基于um.exe的图形化音乐解密工具，支持多种加密音乐格式的批量解密。

## 功能特性

- 🎵 支持多种加密音乐格式：NCM、QMC、KGM、KWM、TM、XM等
- 📁 支持文件和文件夹批量处理
- 🖱️ 支持拖拽文件到界面
- 🔄 多线程并行处理，提高转换效率
- 📊 实时显示转换进度和状态
- 🎯 可自定义输出目录
- 🚀 单文件可执行程序，无需安装

## 支持的音乐格式

| 格式 | 说明 | 来源 |
|------|------|------|
| .ncm | 网易云音乐加密格式 | 网易云音乐 |
| .qmc/.qmc0 | QQ音乐加密格式 | QQ音乐 |
| .qmcflac | QQ音乐FLAC加密格式 | QQ音乐 |
| .qmcogg | QQ音乐OGG加密格式 | QQ音乐 |
| .kgm/.kgma | 酷狗音乐加密格式 | 酷狗音乐 |
| .kwm | 酷我音乐加密格式 | 酷我音乐 |
| .tm | 太合音乐加密格式 | 太合音乐 |
| .xm | 虾米音乐加密格式 | 虾米音乐 |

## 使用方法

### 方法一：直接运行可执行文件
1. 下载 `MusicUnlockGUI.exe`
2. 双击运行程序
3. 选择输出目录
4. 添加要转换的音乐文件（支持拖拽）
5. 点击"开始转换"

### 方法二：从源码运行
1. 确保已安装Python 3.7+
2. 克隆或下载项目源码
3. 确保`um.exe`文件在项目根目录
4. 运行：`python music_unlock_gui/main.py`

## 开发和打包

### 环境要求
- Python 3.7+
- PyInstaller 5.0+
- um.exe（音乐解密核心程序）

### 打包步骤
1. 安装依赖：
   ```bash
   pip install pyinstaller
   ```

2. 确保um.exe在项目根目录

3. 执行打包：
   ```bash
   pyinstaller music_unlock_gui/build_config/build.spec
   ```

4. 打包完成后，可执行文件位于 `dist/MusicUnlockGUI.exe`

### 项目结构
```
music_unlock_gui/
├── main.py              # 主程序入口
├── gui/
│   ├── __init__.py
│   └── main_window.py   # 主界面
├── core/
│   ├── __init__.py
│   ├── processor.py     # 文件处理器
│   └── thread_manager.py # 线程管理器
├── utils/
│   ├── __init__.py
│   └── helpers.py       # 工具函数
├── build_config/
│   └── build.spec       # PyInstaller配置
├── requirements.txt     # 依赖列表
└── README.md           # 说明文档
```

## 注意事项

1. **版权声明**：本工具仅用于个人学习和研究，请勿用于商业用途
2. **文件安全**：建议在转换前备份原始文件
3. **输出目录**：确保输出目录有足够的磁盘空间
4. **文件格式**：转换后的文件格式取决于原始加密文件的内容

## 技术特点

- **多线程处理**：默认使用6个工作线程并行处理文件
- **内存优化**：流式处理大文件，避免内存溢出
- **错误处理**：完善的错误处理和用户提示
- **跨平台**：基于Python和tkinter，支持Windows、macOS、Linux

## 故障排除

### 常见问题

1. **找不到um.exe**
   - 确保um.exe文件与程序在同一目录
   - 检查文件权限

2. **转换失败**
   - 检查输入文件是否为支持的格式
   - 确保输出目录可写
   - 查看错误信息获取详细原因

3. **程序无响应**
   - 大文件转换需要时间，请耐心等待
   - 可以点击"停止转换"取消操作

## 许可证

本项目基于原始um项目的许可证，仅供学习和研究使用。

## 致谢

- 感谢 [unlock-music](https://git.unlock-music.dev/um/cli) 项目提供的核心解密功能
- 感谢所有为音乐解密技术做出贡献的开发者
