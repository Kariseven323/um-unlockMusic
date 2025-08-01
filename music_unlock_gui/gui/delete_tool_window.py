#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
文件删除工具窗口
"""

import tkinter as tk
from tkinter import ttk, filedialog, messagebox
import os
from typing import List, Dict, Set, Tuple
import queue

from core.file_deleter import ThreadedFileDeleter
from core.constants import (
    PLATFORM_FORMAT_GROUPS,
    OUTPUT_FORMATS,
    DELETE_TOOL_WINDOW_TITLE,
    DELETE_TOOL_WINDOW_SIZE,
    DELETE_TOOL_WINDOW_MIN_SIZE,
    DELETE_TOOL_MESSAGES
)
from utils.helpers import center_window


class DeleteToolWindow:
    """文件删除工具窗口类"""
    
    def __init__(self, parent: tk.Tk, supported_extensions: List[str]):
        """
        初始化删除工具窗口
        
        Args:
            parent: 父窗口
            supported_extensions: 支持的文件扩展名列表
        """
        self.parent = parent
        self.supported_extensions = supported_extensions
        self.window = None
        
        # 格式选择状态
        self.encrypted_format_vars = {}  # 加密格式选择状态
        self.output_format_vars = {}     # 输出格式选择状态
        self.platform_vars = {}         # 平台主复选框状态
        
        # 文件操作相关
        self.folder_path = ""
        self.scanned_files = []
        self.file_deleter = ThreadedFileDeleter()
        
        # UI组件
        self.folder_var = None
        self.file_tree = None
        self.progress_var = None
        self.progress_bar = None
        self.status_var = None
        self.scan_button = None
        self.delete_button = None
        self.cancel_button = None
    
    def show(self):
        """显示窗口"""
        if self.window is not None:
            self.window.lift()
            return
        
        self.window = tk.Toplevel(self.parent)
        self.window.title(DELETE_TOOL_WINDOW_TITLE)
        self.window.geometry(DELETE_TOOL_WINDOW_SIZE)
        self.window.minsize(*DELETE_TOOL_WINDOW_MIN_SIZE)
        
        # 设置窗口居中
        self.window.update_idletasks()
        width = self.window.winfo_width()
        height = self.window.winfo_height()
        center_window(self.window, width, height)
        
        # 设置窗口为模态
        self.window.transient(self.parent)
        self.window.grab_set()
        
        # 创建UI
        self.setup_ui()
        
        # 绑定关闭事件
        self.window.protocol("WM_DELETE_WINDOW", self.on_closing)
        
        # 开始检查消息队列
        self.check_queue()

    def _bind_mousewheel_to_widget(self, widget, canvas):
        """为组件绑定鼠标滚轮事件"""
        def _on_mousewheel(event):
            canvas.yview_scroll(int(-1*(event.delta/120)), "units")

        def _bind_to_mousewheel(event):
            canvas.bind_all("<MouseWheel>", _on_mousewheel)

        def _unbind_from_mousewheel(event):
            canvas.unbind_all("<MouseWheel>")

        widget.bind('<Enter>', _bind_to_mousewheel)
        widget.bind('<Leave>', _unbind_from_mousewheel)
    
    def setup_ui(self):
        """设置用户界面"""
        # 主框架
        main_frame = ttk.Frame(self.window, padding="10")
        main_frame.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
        
        # 配置网格权重
        self.window.columnconfigure(0, weight=1)
        self.window.rowconfigure(0, weight=1)
        main_frame.columnconfigure(0, weight=1)
        main_frame.rowconfigure(1, weight=1)
        
        # 创建格式选择区域
        self.create_format_selection(main_frame)
        
        # 创建文件夹选择和文件列表区域
        self.create_file_area(main_frame)
        
        # 创建操作按钮区域
        self.create_button_area(main_frame)
        
        # 创建状态栏
        self.create_status_bar(main_frame)
    
    def create_format_selection(self, parent):
        """创建格式选择区域"""
        format_frame = ttk.LabelFrame(parent, text="文件格式选择", padding="5")
        format_frame.grid(row=0, column=0, sticky=(tk.W, tk.E), pady=(0, 10))
        format_frame.columnconfigure(0, weight=1)
        format_frame.columnconfigure(1, weight=1)
        
        # 双列布局
        left_frame = ttk.LabelFrame(format_frame, text="加密格式（输入）", padding="5")
        left_frame.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S), padx=(0, 5))
        
        right_frame = ttk.LabelFrame(format_frame, text="解密格式（输出）", padding="5")
        right_frame.grid(row=0, column=1, sticky=(tk.W, tk.E, tk.N, tk.S), padx=(5, 0))
        
        # 创建加密格式选择
        self.create_encrypted_format_selection(left_frame)
        
        # 创建输出格式选择
        self.create_output_format_selection(right_frame)
    
    def create_encrypted_format_selection(self, parent):
        """创建加密格式选择区域"""
        # 全选/全不选按钮
        button_frame = ttk.Frame(parent)
        button_frame.grid(row=0, column=0, sticky=(tk.W, tk.E), pady=(0, 5))
        
        ttk.Button(button_frame, text="全选", 
                  command=lambda: self.toggle_all_formats(self.encrypted_format_vars, True)).pack(side=tk.LEFT, padx=(0, 5))
        ttk.Button(button_frame, text="全不选", 
                  command=lambda: self.toggle_all_formats(self.encrypted_format_vars, False)).pack(side=tk.LEFT)
        
        # 创建滚动框架
        canvas = tk.Canvas(parent, height=150)
        scrollbar = ttk.Scrollbar(parent, orient="vertical", command=canvas.yview)
        scrollable_frame = ttk.Frame(canvas)
        
        scrollable_frame.bind(
            "<Configure>",
            lambda e: canvas.configure(scrollregion=canvas.bbox("all"))
        )

        canvas.create_window((0, 0), window=scrollable_frame, anchor="nw")
        canvas.configure(yscrollcommand=scrollbar.set)

        # 绑定鼠标滚轮事件到canvas和scrollable_frame
        def _on_mousewheel(event):
            canvas.yview_scroll(int(-1*(event.delta/120)), "units")

        def _bind_to_mousewheel(event):
            canvas.bind_all("<MouseWheel>", _on_mousewheel)

        def _unbind_from_mousewheel(event):
            canvas.unbind_all("<MouseWheel>")

        # 当鼠标进入canvas区域时绑定滚轮事件
        canvas.bind('<Enter>', _bind_to_mousewheel)
        canvas.bind('<Leave>', _unbind_from_mousewheel)
        scrollable_frame.bind('<Enter>', _bind_to_mousewheel)
        scrollable_frame.bind('<Leave>', _unbind_from_mousewheel)
        
        canvas.grid(row=1, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
        scrollbar.grid(row=1, column=1, sticky=(tk.N, tk.S))
        
        parent.rowconfigure(1, weight=1)
        parent.columnconfigure(0, weight=1)
        
        # 按平台分组显示格式
        row = 0
        for platform, config in PLATFORM_FORMAT_GROUPS.items():
            # 该平台的格式
            keywords = config["keywords"]
            platform_extensions = [ext for ext in self.supported_extensions if
                                 any(keyword in ext for keyword in keywords)]

            if not platform_extensions:
                continue

            # 平台主复选框
            platform_var = tk.BooleanVar()
            self.platform_vars[platform] = platform_var

            platform_cb = ttk.Checkbutton(
                scrollable_frame,
                text=f"📁 {platform} (全选)",
                variable=platform_var,
                command=lambda p=platform: self.toggle_platform_formats(p)
            )
            platform_cb.grid(row=row, column=0, sticky=tk.W, pady=(5, 2))
            # 为平台复选框绑定滚轮事件
            self._bind_mousewheel_to_widget(platform_cb, canvas)
            row += 1

            # 该平台的具体格式
            platform_format_vars = []
            for ext in sorted(platform_extensions):
                var = tk.BooleanVar()
                self.encrypted_format_vars[ext] = var
                platform_format_vars.append(var)

                format_cb = ttk.Checkbutton(
                    scrollable_frame,
                    text=ext,
                    variable=var,
                    command=lambda p=platform: self.update_platform_checkbox(p)
                )
                format_cb.grid(row=row, column=0, sticky=tk.W, padx=(20, 0))
                # 为格式复选框绑定滚轮事件
                self._bind_mousewheel_to_widget(format_cb, canvas)
                row += 1

            # 存储平台对应的格式变量，用于后续操作
            if not hasattr(self, 'platform_format_mapping'):
                self.platform_format_mapping = {}
            self.platform_format_mapping[platform] = platform_format_vars
    
    def create_output_format_selection(self, parent):
        """创建输出格式选择区域"""
        # 全选/全不选按钮
        button_frame = ttk.Frame(parent)
        button_frame.grid(row=0, column=0, sticky=(tk.W, tk.E), pady=(0, 5))
        
        ttk.Button(button_frame, text="全选", 
                  command=lambda: self.toggle_all_formats(self.output_format_vars, True)).pack(side=tk.LEFT, padx=(0, 5))
        ttk.Button(button_frame, text="全不选", 
                  command=lambda: self.toggle_all_formats(self.output_format_vars, False)).pack(side=tk.LEFT)
        
        # 创建滚动框架（与加密格式区域保持一致）
        canvas2 = tk.Canvas(parent, height=150)
        scrollbar2 = ttk.Scrollbar(parent, orient="vertical", command=canvas2.yview)
        format_frame = ttk.Frame(canvas2)

        format_frame.bind(
            "<Configure>",
            lambda e: canvas2.configure(scrollregion=canvas2.bbox("all"))
        )

        canvas2.create_window((0, 0), window=format_frame, anchor="nw")
        canvas2.configure(yscrollcommand=scrollbar2.set)

        canvas2.grid(row=1, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
        scrollbar2.grid(row=1, column=1, sticky=(tk.N, tk.S))

        parent.rowconfigure(1, weight=1)
        parent.columnconfigure(0, weight=1)

        # 绑定鼠标滚轮事件到canvas2和format_frame
        def _on_mousewheel2(event):
            canvas2.yview_scroll(int(-1*(event.delta/120)), "units")

        def _bind_to_mousewheel2(event):
            canvas2.bind_all("<MouseWheel>", _on_mousewheel2)

        def _unbind_from_mousewheel2(event):
            canvas2.unbind_all("<MouseWheel>")

        canvas2.bind('<Enter>', _bind_to_mousewheel2)
        canvas2.bind('<Leave>', _unbind_from_mousewheel2)
        format_frame.bind('<Enter>', _bind_to_mousewheel2)
        format_frame.bind('<Leave>', _unbind_from_mousewheel2)
        
        # 显示输出格式
        row = 0
        for category, config in OUTPUT_FORMATS.items():
            # 分类主复选框
            category_var = tk.BooleanVar()
            if not hasattr(self, 'output_category_vars'):
                self.output_category_vars = {}
            self.output_category_vars[category] = category_var

            category_cb = ttk.Checkbutton(
                format_frame,
                text=f"📁 {category} (全选)",
                variable=category_var,
                command=lambda c=category: self.toggle_output_category_formats(c)
            )
            category_cb.grid(row=row, column=0, sticky=tk.W, pady=(5, 2))
            # 为分类复选框绑定滚轮事件
            self._bind_mousewheel_to_widget(category_cb, canvas2)
            row += 1

            # 格式选项
            category_format_vars = []
            for fmt in config["formats"]:
                var = tk.BooleanVar()
                self.output_format_vars[fmt] = var
                category_format_vars.append(var)

                format_cb = ttk.Checkbutton(
                    format_frame,
                    text=fmt,
                    variable=var,
                    command=lambda c=category: self.update_output_category_checkbox(c)
                )
                format_cb.grid(row=row, column=0, sticky=tk.W, padx=(20, 0))
                # 为格式复选框绑定滚轮事件
                self._bind_mousewheel_to_widget(format_cb, canvas2)
                row += 1

            # 存储分类对应的格式变量
            if not hasattr(self, 'output_category_format_mapping'):
                self.output_category_format_mapping = {}
            self.output_category_format_mapping[category] = category_format_vars
    
    def toggle_all_formats(self, format_vars: Dict[str, tk.BooleanVar], select_all: bool):
        """切换所有格式的选择状态"""
        for var in format_vars.values():
            var.set(select_all)

        # 同时更新平台/分类复选框状态
        if format_vars == self.encrypted_format_vars:
            for platform in self.platform_vars:
                self.update_platform_checkbox(platform)
        elif format_vars == self.output_format_vars:
            if hasattr(self, 'output_category_vars'):
                for category in self.output_category_vars:
                    self.update_output_category_checkbox(category)

    def toggle_platform_formats(self, platform: str):
        """切换指定平台的所有格式选择状态"""
        if platform not in self.platform_vars or platform not in self.platform_format_mapping:
            return

        platform_selected = self.platform_vars[platform].get()
        platform_format_vars = self.platform_format_mapping[platform]

        # 设置该平台下所有格式的状态
        for var in platform_format_vars:
            var.set(platform_selected)

    def update_platform_checkbox(self, platform: str):
        """更新平台复选框状态（基于子格式的选择状态）"""
        if platform not in self.platform_vars or platform not in self.platform_format_mapping:
            return

        platform_format_vars = self.platform_format_mapping[platform]

        # 检查该平台下所有格式的选择状态
        selected_count = sum(1 for var in platform_format_vars if var.get())
        total_count = len(platform_format_vars)

        # 更新平台复选框状态
        if selected_count == 0:
            # 没有选中任何格式
            self.platform_vars[platform].set(False)
        elif selected_count == total_count:
            # 全部选中
            self.platform_vars[platform].set(True)
        else:
            # 部分选中 - 这里我们保持当前状态，或者可以设置为False
            # 为了更好的用户体验，我们设置为False，表示未完全选中
            self.platform_vars[platform].set(False)

    def toggle_output_category_formats(self, category: str):
        """切换指定输出格式分类的所有格式选择状态"""
        if not hasattr(self, 'output_category_vars') or category not in self.output_category_vars:
            return
        if not hasattr(self, 'output_category_format_mapping') or category not in self.output_category_format_mapping:
            return

        category_selected = self.output_category_vars[category].get()
        category_format_vars = self.output_category_format_mapping[category]

        # 设置该分类下所有格式的状态
        for var in category_format_vars:
            var.set(category_selected)

    def update_output_category_checkbox(self, category: str):
        """更新输出格式分类复选框状态（基于子格式的选择状态）"""
        if not hasattr(self, 'output_category_vars') or category not in self.output_category_vars:
            return
        if not hasattr(self, 'output_category_format_mapping') or category not in self.output_category_format_mapping:
            return

        category_format_vars = self.output_category_format_mapping[category]

        # 检查该分类下所有格式的选择状态
        selected_count = sum(1 for var in category_format_vars if var.get())
        total_count = len(category_format_vars)

        # 更新分类复选框状态
        if selected_count == 0:
            # 没有选中任何格式
            self.output_category_vars[category].set(False)
        elif selected_count == total_count:
            # 全部选中
            self.output_category_vars[category].set(True)
        else:
            # 部分选中
            self.output_category_vars[category].set(False)
    
    def create_file_area(self, parent):
        """创建文件区域"""
        file_frame = ttk.LabelFrame(parent, text="文件夹和文件列表", padding="5")
        file_frame.grid(row=1, column=0, sticky=(tk.W, tk.E, tk.N, tk.S), pady=(0, 10))
        file_frame.columnconfigure(0, weight=1)
        file_frame.rowconfigure(1, weight=1)
        
        # 文件夹选择
        folder_frame = ttk.Frame(file_frame)
        folder_frame.grid(row=0, column=0, sticky=(tk.W, tk.E), pady=(0, 5))
        folder_frame.columnconfigure(1, weight=1)
        
        ttk.Label(folder_frame, text="目标文件夹:").grid(row=0, column=0, sticky=tk.W, padx=(0, 5))
        
        self.folder_var = tk.StringVar()
        ttk.Entry(folder_frame, textvariable=self.folder_var, state="readonly").grid(
            row=0, column=1, sticky=(tk.W, tk.E), padx=(0, 5))
        
        ttk.Button(folder_frame, text="浏览", command=self.select_folder).grid(row=0, column=2)
        
        # 文件列表
        list_frame = ttk.Frame(file_frame)
        list_frame.grid(row=1, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
        list_frame.columnconfigure(0, weight=1)
        list_frame.rowconfigure(0, weight=1)
        
        # 创建Treeview
        columns = ("文件名", "大小", "路径")
        self.file_tree = ttk.Treeview(list_frame, columns=columns, show="tree headings", height=10)
        
        # 设置列
        self.file_tree.heading("#0", text="序号")
        self.file_tree.heading("文件名", text="文件名")
        self.file_tree.heading("大小", text="大小")
        self.file_tree.heading("路径", text="路径")
        
        self.file_tree.column("#0", width=50, minwidth=50)
        self.file_tree.column("文件名", width=200, minwidth=150)
        self.file_tree.column("大小", width=80, minwidth=80)
        self.file_tree.column("路径", width=300, minwidth=200)
        
        # 滚动条
        scrollbar_y = ttk.Scrollbar(list_frame, orient="vertical", command=self.file_tree.yview)
        scrollbar_x = ttk.Scrollbar(list_frame, orient="horizontal", command=self.file_tree.xview)
        self.file_tree.configure(yscrollcommand=scrollbar_y.set, xscrollcommand=scrollbar_x.set)
        
        self.file_tree.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
        scrollbar_y.grid(row=0, column=1, sticky=(tk.N, tk.S))
        scrollbar_x.grid(row=1, column=0, sticky=(tk.W, tk.E))
    
    def create_button_area(self, parent):
        """创建按钮区域"""
        button_frame = ttk.Frame(parent)
        button_frame.grid(row=2, column=0, sticky=(tk.W, tk.E), pady=(0, 10))
        
        self.scan_button = ttk.Button(button_frame, text="扫描文件", command=self.scan_files)
        self.scan_button.pack(side=tk.LEFT, padx=(0, 5))
        
        self.delete_button = ttk.Button(button_frame, text="删除文件", command=self.delete_files, state="disabled")
        self.delete_button.pack(side=tk.LEFT, padx=(0, 5))
        
        self.cancel_button = ttk.Button(button_frame, text="取消操作", command=self.cancel_operation, state="disabled")
        self.cancel_button.pack(side=tk.LEFT)
    
    def create_status_bar(self, parent):
        """创建状态栏"""
        status_frame = ttk.Frame(parent)
        status_frame.grid(row=3, column=0, sticky=(tk.W, tk.E))
        status_frame.columnconfigure(1, weight=1)
        
        # 状态文本
        ttk.Label(status_frame, text="状态:").grid(row=0, column=0, sticky=tk.W)
        self.status_var = tk.StringVar(value="就绪")
        ttk.Label(status_frame, textvariable=self.status_var).grid(row=0, column=1, sticky=tk.W, padx=(5, 0))
        
        # 进度条
        self.progress_var = tk.StringVar(value="")
        ttk.Label(status_frame, textvariable=self.progress_var).grid(row=1, column=0, columnspan=2, sticky=tk.W, pady=(5, 0))
        
        self.progress_bar = ttk.Progressbar(status_frame, mode='determinate')
        self.progress_bar.grid(row=2, column=0, columnspan=2, sticky=(tk.W, tk.E), pady=(2, 0))

    def select_folder(self):
        """选择文件夹"""
        folder = filedialog.askdirectory(title="选择要处理的文件夹")
        if folder:
            self.folder_path = folder
            self.folder_var.set(folder)
            # 清空文件列表
            self.clear_file_list()
            self.delete_button.config(state="disabled")

    def get_selected_formats(self) -> Tuple[List[str], List[str]]:
        """获取选中的格式"""
        encrypted_formats = [fmt for fmt, var in self.encrypted_format_vars.items() if var.get()]
        output_formats = [fmt for fmt, var in self.output_format_vars.items() if var.get()]
        return encrypted_formats, output_formats

    def scan_files(self):
        """扫描文件"""
        # 验证输入
        if not self.folder_path:
            messagebox.showerror("错误", DELETE_TOOL_MESSAGES['no_folder_selected'])
            return

        if not os.path.exists(self.folder_path):
            messagebox.showerror("错误", DELETE_TOOL_MESSAGES['folder_not_exists'])
            return

        encrypted_formats, output_formats = self.get_selected_formats()
        all_formats = encrypted_formats + output_formats

        if not all_formats:
            messagebox.showerror("错误", DELETE_TOOL_MESSAGES['no_formats_selected'])
            return

        # 清空文件列表
        self.clear_file_list()

        # 更新UI状态
        self.scan_button.config(state="disabled")
        self.delete_button.config(state="disabled")
        self.cancel_button.config(state="normal")
        self.status_var.set(DELETE_TOOL_MESSAGES['scan_started'])
        self.progress_bar['value'] = 0

        # 开始异步扫描
        if not self.file_deleter.scan_files_async(self.folder_path, all_formats):
            messagebox.showerror("错误", "无法启动扫描，可能有其他操作正在进行")
            self.reset_ui_state()

    def delete_files(self):
        """删除文件"""
        if not self.scanned_files:
            messagebox.showwarning("警告", DELETE_TOOL_MESSAGES['no_files_found'])
            return

        # 确认删除
        confirm_msg = DELETE_TOOL_MESSAGES['confirm_delete'].format(len(self.scanned_files))
        if not messagebox.askyesno("确认删除", confirm_msg):
            return

        # 更新UI状态
        self.scan_button.config(state="disabled")
        self.delete_button.config(state="disabled")
        self.cancel_button.config(state="normal")
        self.status_var.set(DELETE_TOOL_MESSAGES['delete_started'])
        self.progress_bar['value'] = 0

        # 开始异步删除
        if not self.file_deleter.delete_files_async(self.scanned_files):
            messagebox.showerror("错误", "无法启动删除，可能有其他操作正在进行")
            self.reset_ui_state()

    def cancel_operation(self):
        """取消操作"""
        self.file_deleter.cancel_operation()
        self.status_var.set("正在取消操作...")

    def clear_file_list(self):
        """清空文件列表"""
        for item in self.file_tree.get_children():
            self.file_tree.delete(item)
        self.scanned_files = []

    def update_file_list(self, files: List[str]):
        """更新文件列表"""
        self.clear_file_list()
        self.scanned_files = files

        for i, file_path in enumerate(files, 1):
            filename = os.path.basename(file_path)
            try:
                size = os.path.getsize(file_path)
                size_str = self.format_file_size(size)
            except:
                size_str = "未知"

            self.file_tree.insert("", "end", text=str(i), values=(filename, size_str, file_path))

    def format_file_size(self, size_bytes: int) -> str:
        """格式化文件大小"""
        if size_bytes == 0:
            return "0 B"

        size_names = ["B", "KB", "MB", "GB"]
        i = 0
        while size_bytes >= 1024 and i < len(size_names) - 1:
            size_bytes /= 1024.0
            i += 1

        return f"{size_bytes:.1f} {size_names[i]}"

    def reset_ui_state(self):
        """重置UI状态"""
        self.scan_button.config(state="normal")
        self.delete_button.config(state="normal" if self.scanned_files else "disabled")
        self.cancel_button.config(state="disabled")
        self.progress_bar['value'] = 0
        self.progress_var.set("")

    def check_queue(self):
        """检查消息队列"""
        try:
            while True:
                message = self.file_deleter.get_message_queue().get_nowait()
                self.handle_message(message)
        except queue.Empty:
            pass

        # 继续检查
        if self.window:
            self.window.after(100, self.check_queue)

    def handle_message(self, message):
        """处理消息"""
        msg_type = message[0]

        if msg_type == 'progress':
            _, text, progress = message
            self.progress_var.set(text)
            self.progress_bar['value'] = progress

        elif msg_type == 'scan_complete':
            _, success, files, error = message
            if success:
                self.update_file_list(files)
                self.status_var.set(f"{DELETE_TOOL_MESSAGES['scan_completed']} - 找到 {len(files)} 个文件")
                self.delete_button.config(state="normal" if files else "disabled")
            else:
                messagebox.showerror("扫描失败", error)
                self.status_var.set("扫描失败")
            self.reset_ui_state()

        elif msg_type == 'delete_complete':
            _, deleted, failed, failed_files = message
            result_msg = f"删除完成：成功 {deleted} 个，失败 {failed} 个"
            self.status_var.set(result_msg)

            if failed > 0:
                # 显示失败的文件
                failed_msg = f"以下 {failed} 个文件删除失败：\n\n" + "\n".join(failed_files[:10])
                if failed > 10:
                    failed_msg += f"\n... 还有 {failed - 10} 个文件"
                messagebox.showwarning("删除结果", failed_msg)
            else:
                messagebox.showinfo("删除完成", "所有文件已成功删除！")

            # 清空文件列表并重置状态
            self.clear_file_list()
            self.reset_ui_state()

    def on_closing(self):
        """窗口关闭事件"""
        if self.file_deleter.is_busy():
            if messagebox.askyesno("确认关闭", "正在执行操作，确定要关闭窗口吗？"):
                self.file_deleter.cancel_operation()
                self.window.destroy()
                self.window = None
        else:
            self.window.destroy()
            self.window = None


class DeleteToolEmbedded(DeleteToolWindow):
    """内嵌版本的文件删除工具"""

    def __init__(self, parent_frame: ttk.Frame, supported_extensions: List[str]):
        """
        初始化内嵌删除工具

        Args:
            parent_frame: 父框架
            supported_extensions: 支持的文件扩展名列表
        """
        # 不调用父类的__init__，而是直接初始化需要的属性
        self.parent_frame = parent_frame
        self.supported_extensions = supported_extensions
        self.window = None  # 内嵌版本不需要窗口

        # 格式选择状态
        self.encrypted_format_vars = {}
        self.output_format_vars = {}
        self.platform_vars = {}

        # 文件操作相关
        self.folder_path = ""
        self.scanned_files = []
        self.file_deleter = ThreadedFileDeleter()

        # UI组件
        self.folder_var = None
        self.file_tree = None
        self.progress_var = None
        self.progress_bar = None
        self.status_var = None
        self.scan_button = None
        self.delete_button = None
        self.cancel_button = None

        # 创建UI
        self.setup_embedded_ui()

        # 开始检查消息队列
        self.check_queue()

    def setup_embedded_ui(self):
        """设置内嵌UI"""
        # 配置网格权重
        self.parent_frame.columnconfigure(0, weight=1)
        self.parent_frame.rowconfigure(1, weight=1)

        # 创建格式选择区域
        self.create_format_selection(self.parent_frame)

        # 创建文件夹选择和文件列表区域
        self.create_file_area(self.parent_frame)

        # 创建操作按钮区域
        self.create_button_area(self.parent_frame)

        # 创建状态栏
        self.create_status_bar(self.parent_frame)

    def check_queue(self):
        """检查消息队列（内嵌版本）"""
        try:
            while True:
                message = self.file_deleter.get_message_queue().get_nowait()
                self.handle_message(message)
        except queue.Empty:
            pass

        # 继续检查（使用parent_frame的after方法）
        if self.parent_frame:
            self.parent_frame.after(100, self.check_queue)
