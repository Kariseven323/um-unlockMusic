#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
音乐解密GUI工具 - 开发环境启动脚本
"""

import sys
import os

# 添加项目根目录到Python路径
current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, current_dir)

# 导入并运行主程序
from main import main

if __name__ == "__main__":
    main()
