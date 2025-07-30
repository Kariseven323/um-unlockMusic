#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
音乐解密GUI工具 - 主程序入口
基于um.exe的图形化界面
"""

import sys
import os
import tkinter as tk
from tkinter import messagebox

# 添加项目路径到sys.path
current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, current_dir)

from gui.main_window import MusicUnlockGUI


def get_um_exe_path():
    """获取um.exe的路径，支持打包后的环境"""
    if getattr(sys, 'frozen', False):
        # 打包后的环境
        base_path = sys._MEIPASS
    else:
        # 开发环境
        base_path = os.path.dirname(os.path.abspath(__file__))
    
    um_exe_path = os.path.join(base_path, 'um.exe')
    
    # 如果在开发环境中找不到，尝试在上级目录查找
    if not os.path.exists(um_exe_path):
        parent_dir = os.path.dirname(current_dir)
        um_exe_path = os.path.join(parent_dir, 'um.exe')
    
    return um_exe_path


def main():
    """主函数"""
    try:
        # 检查um.exe是否存在
        um_exe_path = get_um_exe_path()
        if not os.path.exists(um_exe_path):
            messagebox.showerror(
                "错误", 
                f"找不到um.exe文件！\n期望路径：{um_exe_path}\n\n请确保um.exe文件存在。"
            )
            return
        
        # 创建主窗口
        root = tk.Tk()
        app = MusicUnlockGUI(root, um_exe_path)
        
        # 设置窗口关闭事件
        def on_closing():
            if app.is_processing():
                if messagebox.askokcancel("退出", "正在处理文件，确定要退出吗？"):
                    app.stop_all_tasks()
                    root.destroy()
            else:
                root.destroy()
        
        root.protocol("WM_DELETE_WINDOW", on_closing)
        
        # 启动GUI
        root.mainloop()
        
    except Exception as e:
        messagebox.showerror("启动错误", f"程序启动失败：{str(e)}")
        sys.exit(1)


if __name__ == "__main__":
    main()
