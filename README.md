# 🎵 Unlock Music - 音乐解密工具

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.23.3-blue.svg)](https://golang.org/)
[![Python Version](https://img.shields.io/badge/Python-3.7+-blue.svg)](https://www.python.org/)
[![GitHub Release](https://img.shields.io/github/v/release/Kariseven323/um-unlockMusic)](https://github.com/Kariseven323/um-unlockMusic/releases)

> **项目来源声明**
> 本项目基于 [Unlock Music CLI](https://git.unlock-music.dev/um/cli) 开发，在原有 CLI 工具基础上增加了图形化界面和性能优化功能。

一个功能强大的音乐解密工具，支持多种主流音乐平台的加密格式解密，提供命令行和图形化两种使用方式。

## ✨ 功能特性

- 🎵 **多格式支持**：支持 NCM、QMC、KGM、KWM、TM、XM 等主流加密格式
- 🖥️ **双重界面**：提供 CLI 命令行和 GUI 图形化两种操作方式
- 📁 **批量处理**：支持文件夹批量解密，高效处理大量文件
- 🚀 **性能优化**：采用批处理模式，相比传统方式性能提升 60-80%
- 🎯 **智能命名**：支持多种文件命名格式，自动提取音乐元数据
- 🔄 **多线程处理**：充分利用多核 CPU，提高处理效率
- 📊 **实时进度**：GUI 界面实时显示处理进度和状态
- 🎨 **用户友好**：支持拖拽操作，简单易用

## 📋 支持格式

| 格式 | 说明 | 来源平台 |
|------|------|----------|
| `.ncm` | 网易云音乐加密格式 | 网易云音乐 |
| `.qmc/.qmc0` | QQ音乐加密格式 | QQ音乐 |
| `.qmcflac` | QQ音乐FLAC加密格式 | QQ音乐 |
| `.qmcogg` | QQ音乐OGG加密格式 | QQ音乐 |
| `.kgm/.kgma` | 酷狗音乐加密格式 | 酷狗音乐 |
| `.kwm` | 酷我音乐加密格式 | 酷我音乐 |
| `.tm` | 太合音乐加密格式 | 太合音乐 |
| `.xm` | 虾米音乐加密格式 | 虾米音乐 |

## 🚀 快速开始

### 方式一：下载预编译版本（推荐）

1. 前往 [Releases](https://github.com/Kariseven323/um-unlockMusic/releases) 页面
2. 下载最新版本的 `MusicUnlockGUI.exe`
3. 双击运行，开始使用图形化界面

### 方式二：使用命令行工具

```bash
# 解密单个文件
./um.exe -i input.ncm -o output/

# 批量解密文件夹
./um.exe -i input_folder/ -o output_folder/

# 使用批处理模式（高性能）
echo '{"files":[{"input_path":"test.ncm"}],"options":{"update_metadata":true}}' | ./um.exe --batch
```

## 🛠️ 从源码构建

### 环境要求

- **Go**: 1.23.3 或更高版本
- **Python**: 3.7+ （仅 GUI 需要）

### 构建步骤

1. **克隆仓库**
   ```bash
   git clone https://github.com/Kariseven323/um-unlockMusic.git
   cd um-unlockMusic
   ```

2. **构建 CLI 工具**
   ```bash
   go build -o um.exe ./cmd/um
   ```

3. **运行 GUI 界面**
   ```bash
   python music_unlock_gui/main.py
   ```

4. **打包 GUI 为可执行文件**
   ```bash
   # Windows
   ./build.bat

   # 或手动打包
   pip install pyinstaller
   pyinstaller music_unlock_gui/build_config/build.spec
   ```

## 📖 使用说明

### GUI 图形界面

1. **启动程序**：双击 `MusicUnlockGUI.exe`
2. **选择输出目录**：点击"选择输出目录"按钮
3. **添加文件**：
   - 点击"添加文件"选择单个文件
   - 点击"添加文件夹"选择整个文件夹

4. **开始转换**：点击"开始转换"按钮
5. **查看进度**：实时查看转换进度和结果

### CLI 命令行

```bash
# 基本用法
um [选项] -i 输入文件/文件夹 -o 输出目录

# 常用选项
-i, --input          输入文件或目录
-o, --output         输出目录
-r, --remove-source  转换后删除源文件
-f, --overwrite      覆盖已存在的文件
-w, --watch          监视目录变化
-j, --workers        工作线程数 (默认: 1)
--batch              批处理模式 (高性能)
--naming-format      文件命名格式
-v, --verbose        详细输出
-h, --help           显示帮助
```

### 批处理模式（高性能）

```bash
# JSON 格式输入
echo '{
  "files": [
    {"input_path": "song1.ncm"},
    {"input_path": "song2.qmc", "output_path": "custom_output/"}
  ],
  "options": {
    "update_metadata": true,
    "overwrite_output": true,
    "naming_format": "auto"
  }
}' | um.exe --batch
```

## 🏗️ 项目架构

```
um-unlockMusic/
├── cmd/um/                 # Go CLI 主程序
├── algo/                   # 解密算法实现
│   ├── ncm/               # 网易云音乐
│   ├── qmc/               # QQ音乐
│   ├── kgm/               # 酷狗音乐
│   └── ...                # 其他格式
├── music_unlock_gui/       # Python GUI 界面
│   ├── gui/               # 界面组件
│   ├── core/              # 核心处理逻辑
│   └── utils/             # 工具函数
├── internal/              # 内部工具库
└── dist/                  # 构建输出
```

## ⚡ 性能优化

本项目实现了多项性能优化：

- **批处理模式**：一次调用处理多个文件，减少 90% 进程启动开销
- **多线程并行**：充分利用多核 CPU 性能
- **内存池优化**：减少内存分配和垃圾回收
- **流式处理**：支持大文件的流式解密

相比传统逐文件处理方式，整体性能提升 **60-80%**。

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## ⚠️ 免责声明

- 本工具仅供个人学习和研究使用
- 请勿用于商业用途或侵犯版权
- 使用本工具产生的任何法律问题与开发者无关
- 建议在使用前备份原始文件

## 🔗 相关链接

- [原项目地址](https://git.unlock-music.dev/um/cli)
- [问题反馈](https://github.com/Kariseven323/um-unlockMusic/issues)
- [发布页面](https://github.com/Kariseven323/um-unlockMusic/releases)

---

**如果这个项目对您有帮助，请给个 ⭐ Star 支持一下！**
