#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
文件处理器 - 负责调用um.exe处理音乐文件
"""

import os
import subprocess
import logging
import platform
from typing import Optional, Tuple, List, Dict, Any
import tempfile
import shutil

from .constants import (
    DEFAULT_SUPPORTED_EXTENSIONS,
    PROCESS_TIMEOUT_SECONDS,
    UM_COMMAND_TIMEOUT,
    LOG_FORMAT,
    ERROR_MESSAGES,
    SUCCESS_MESSAGES
)
from .service_client import ServiceClient


class FileProcessor:
    """文件处理器类"""

    def __init__(self, um_exe_path: str, use_service_mode: bool = True):
        """
        初始化文件处理器

        Args:
            um_exe_path: um.exe的路径
            use_service_mode: 是否使用服务模式
        """
        self.um_exe_path = um_exe_path
        self.use_service_mode = use_service_mode
        self.service_client = None
        self.service_available = False
        self.logger = self._setup_logger()

        # 验证um.exe是否存在
        if not os.path.exists(um_exe_path):
            raise FileNotFoundError(ERROR_MESSAGES['um_exe_not_found'].format(um_exe_path))

        # 获取支持的格式列表
        success, self.supported_extensions = self.get_supported_extensions()
        if success:
            self.logger.info(f"成功获取 {len(self.supported_extensions)} 个支持的格式")
        else:
            self.logger.warning("使用默认格式列表")

        # 转换为集合以提高查找效率
        self.supported_extensions_set = {ext.lower() for ext in self.supported_extensions}

        # 初始化服务模式
        if self.use_service_mode:
            self._init_service_mode()

        # 会话复用（避免每次创建新会话）
        self._persistent_session = False

    def _init_service_mode(self):
        """初始化服务模式"""
        try:
            self.service_client = ServiceClient()
            # 尝试连接服务，如果失败则启动服务
            if not self.service_client.connect(timeout=2.0):
                self.logger.info("服务未运行，尝试启动服务")
                if self._start_service():
                    # 等待服务启动
                    import time
                    time.sleep(1)
                    if self.service_client.connect(timeout=5.0):
                        self.service_available = True
                        self.logger.info("服务模式已启用")
                    else:
                        self.logger.warning("服务启动后仍无法连接，将使用传统模式")
                        self.service_available = False
                else:
                    self.logger.warning("无法启动服务，将使用传统模式")
                    self.service_available = False
            else:
                self.service_available = True
                self.logger.info("服务模式已启用")
        except Exception as e:
            self.logger.warning(f"初始化服务模式失败: {e}，将使用传统模式")
            self.service_available = False

    def _start_service(self) -> bool:
        """启动服务"""
        try:
            import subprocess
            # 启动服务进程
            cmd = [self.um_exe_path, "--service"]
            subprocess.Popen(cmd,
                           stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL,
                           creationflags=subprocess.CREATE_NO_WINDOW if platform.system() == "Windows" else 0)
            return True
        except Exception as e:
            self.logger.error(f"启动服务失败: {e}")
            return False

    def is_service_available(self) -> bool:
        """检查服务是否可用"""
        return self.service_available and self.service_client is not None

    def _setup_logger(self) -> logging.Logger:
        """设置日志记录器"""
        logger = logging.getLogger('FileProcessor')
        logger.setLevel(logging.INFO)

        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter(LOG_FORMAT)
            handler.setFormatter(formatter)
            logger.addHandler(handler)

        return logger

    def _get_subprocess_kwargs(self) -> dict:
        """
        获取subprocess调用的参数，在Windows下隐藏控制台窗口

        Returns:
            dict: subprocess.run()的额外参数
        """
        kwargs = {}
        if platform.system() == 'Windows':
            # 隐藏控制台窗口
            kwargs['creationflags'] = subprocess.CREATE_NO_WINDOW
        return kwargs

    def is_supported_file(self, file_path: str) -> bool:
        """
        检查文件是否为支持的音乐格式

        Args:
            file_path: 文件路径

        Returns:
            bool: 是否支持
        """
        file_ext = os.path.splitext(file_path)[1].lower()

        # 使用动态获取的支持格式列表
        return file_ext in self.supported_extensions_set
    
    def process_file(self, input_file: str, output_dir: str = None,
                    progress_callback=None, use_source_dir: bool = False,
                    naming_format: str = "auto") -> Tuple[bool, str]:
        """
        处理单个音乐文件

        Args:
            input_file: 输入文件路径
            output_dir: 输出目录路径（可选）
            progress_callback: 进度回调函数
            use_source_dir: 是否使用源文件目录作为输出目录
            naming_format: 文件命名格式 (auto, title-artist, artist-title, original)

        Returns:
            Tuple[bool, str]: (是否成功, 错误信息或成功信息)
        """
        try:
            # 检查输入文件是否存在
            if not os.path.exists(input_file):
                return False, ERROR_MESSAGES['file_not_found'].format(input_file)

            # 检查文件格式是否支持
            if not self.is_supported_file(input_file):
                return False, ERROR_MESSAGES['unsupported_format'].format(os.path.splitext(input_file)[1])

            # 确定实际输出目录
            actual_output_dir = self._determine_output_dir(input_file, output_dir, use_source_dir)

            # 确保输出目录存在
            os.makedirs(actual_output_dir, exist_ok=True)
            
            # 构建命令行参数
            cmd = [
                self.um_exe_path,
                '-i', input_file,
                '-o', actual_output_dir,
                '--naming-format', naming_format,
                '--overwrite',  # 覆盖已存在的文件
                '--verbose'     # 详细输出
            ]
            
            self.logger.info(f"开始处理文件: {input_file}")
            self.logger.debug(f"执行命令: {' '.join(cmd)}")
            
            # 调用进度回调
            if progress_callback:
                progress_callback(10)  # 开始处理
            
            # 执行um.exe
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                encoding='utf-8',
                errors='ignore',
                timeout=PROCESS_TIMEOUT_SECONDS,
                **self._get_subprocess_kwargs()
            )
            
            # 调用进度回调
            if progress_callback:
                progress_callback(90)  # 处理完成
            
            # 检查执行结果
            if result.returncode == 0:
                success_msg = SUCCESS_MESSAGES['conversion_success'].format(os.path.basename(input_file))
                self.logger.info(success_msg)
                
                if progress_callback:
                    progress_callback(100)  # 完成
                
                return True, success_msg
            else:
                error_msg = ERROR_MESSAGES['conversion_failed'].format(result.stderr or result.stdout or '未知错误')
                self.logger.error(f"处理文件失败: {input_file}, 错误: {error_msg}")
                return False, error_msg

        except subprocess.TimeoutExpired:
            error_msg = ERROR_MESSAGES['processing_timeout']
            self.logger.error(f"处理文件超时: {input_file}")
            return False, error_msg

        except Exception as e:
            error_msg = ERROR_MESSAGES['processing_exception'].format(str(e))
            self.logger.error(f"处理文件异常: {input_file}, 错误: {error_msg}")
            return False, error_msg
    
    def get_output_filename(self, input_file: str) -> str:
        """
        根据输入文件名推测输出文件名
        
        Args:
            input_file: 输入文件路径
            
        Returns:
            str: 预期的输出文件名（不含路径）
        """
        base_name = os.path.splitext(os.path.basename(input_file))[0]
        
        # 根据输入格式推测输出格式
        input_ext = os.path.splitext(input_file)[1].lower()
        
        if input_ext in ['.ncm']:
            # 网易云音乐通常输出为mp3或flac
            return f"{base_name}.mp3"  # 默认假设为mp3
        elif input_ext in ['.qmcflac']:
            return f"{base_name}.flac"
        elif input_ext in ['.qmcogg']:
            return f"{base_name}.ogg"
        elif input_ext in ['.qmc', '.qmc0']:
            return f"{base_name}.mp3"  # 默认假设为mp3
        elif input_ext in ['.kgm', '.kgma']:
            return f"{base_name}.mp3"  # 默认假设为mp3
        elif input_ext in ['.kwm']:
            return f"{base_name}.mp3"  # 默认假设为mp3
        else:
            return f"{base_name}.mp3"  # 默认输出格式
    
    def validate_um_exe(self) -> Tuple[bool, str]:
        """
        验证um.exe是否可用
        
        Returns:
            Tuple[bool, str]: (是否可用, 版本信息或错误信息)
        """
        try:
            result = subprocess.run(
                [self.um_exe_path, '--version'],
                capture_output=True,
                text=True,
                encoding='utf-8',
                errors='ignore',
                timeout=10,
                **self._get_subprocess_kwargs()
            )
            
            if result.returncode == 0:
                version_info = result.stdout.strip()
                return True, version_info
            else:
                return False, f"um.exe执行失败: {result.stderr}"
                
        except subprocess.TimeoutExpired:
            return False, "um.exe响应超时"
        except Exception as e:
            return False, f"验证um.exe时出错: {str(e)}"
    
    def get_supported_extensions(self) -> Tuple[bool, list]:
        """
        获取um.exe支持的文件扩展名

        Returns:
            Tuple[bool, list]: (是否成功, 扩展名列表)
        """
        try:
            result = subprocess.run(
                [self.um_exe_path, '--supported-ext'],
                capture_output=True,
                text=True,
                encoding='utf-8',
                errors='ignore',
                timeout=UM_COMMAND_TIMEOUT,
                **self._get_subprocess_kwargs()
            )

            if result.returncode == 0:
                # 解析输出获取支持的扩展名
                # 输出格式为 "扩展名: 数量"，需要提取扩展名部分
                extensions = []
                for line in result.stdout.split('\n'):
                    line = line.strip()
                    if line and ':' in line:
                        ext = line.split(':')[0].strip()
                        if ext:
                            # 确保扩展名以点开头
                            if not ext.startswith('.'):
                                ext = '.' + ext
                            extensions.append(ext)

                self.logger.info(f"从um.exe获取到 {len(extensions)} 个支持的格式")
                return True, extensions
            else:
                self.logger.warning(f"um.exe --supported-ext 命令失败: {result.stderr}")
                return False, self._get_default_extensions()

        except Exception as e:
            self.logger.warning(f"获取支持的扩展名失败: {str(e)}")
            return False, self._get_default_extensions()

    def _get_default_extensions(self) -> list:
        """
        获取默认的支持扩展名列表（包含完整格式）

        Returns:
            list: 默认扩展名列表
        """
        return DEFAULT_SUPPORTED_EXTENSIONS.copy()

    def _determine_output_dir(self, input_file: str, output_dir: str = None,
                             use_source_dir: bool = False) -> str:
        """
        确定实际的输出目录

        Args:
            input_file: 输入文件路径
            output_dir: 用户指定的输出目录
            use_source_dir: 是否使用源文件目录

        Returns:
            str: 实际输出目录路径
        """
        if use_source_dir or output_dir is None:
            # 使用源文件所在目录
            return os.path.dirname(os.path.abspath(input_file))
        else:
            # 使用用户指定的目录
            return output_dir

    def process_files_batch(self, file_list: list, output_dir: str = None,
                           use_source_dir: bool = False, naming_format: str = "auto") -> dict:
        """
        批量处理多个音乐文件（智能选择最优模式）

        Args:
            file_list: 要处理的文件列表
            output_dir: 输出目录路径（可选）
            use_source_dir: 是否使用源文件目录作为输出目录
            naming_format: 文件命名格式 (auto, title-artist, artist-title, original)

        Returns:
            dict: 批处理结果
        """
        # 智能模式选择：
        # - 文件数量 >= 50 或者服务已经运行：使用服务模式
        # - 文件数量 < 50 且服务未运行：使用传统模式（避免启动开销）
        file_count = len(file_list)

        if self.is_service_available() and file_count >= 10:
            # 服务可用且文件数量足够多，使用服务模式
            return self._process_files_batch_service(file_list, output_dir, use_source_dir, naming_format)
        elif self.use_service_mode and file_count >= 50:
            # 文件数量很多，值得启动服务
            self._init_service_mode()  # 重新尝试启动服务
            if self.is_service_available():
                return self._process_files_batch_service(file_list, output_dir, use_source_dir, naming_format)

        # 使用传统模式
        return self._process_files_batch_subprocess(file_list, output_dir, use_source_dir, naming_format)

    def _process_files_batch_service(self, file_list: list, output_dir: str = None,
                                   use_source_dir: bool = False, naming_format: str = "auto") -> dict:
        """
        使用服务模式批量处理文件
        """
        try:
            # 启动会话
            if not self.service_client.start_session():
                self.logger.error("启动服务会话失败，回退到传统模式")
                return self._process_files_batch_subprocess(file_list, output_dir, use_source_dir, naming_format)

            # 准备文件列表
            files = []
            for file_path in file_list:
                task = {"input_path": file_path}
                if not use_source_dir and output_dir:
                    task["output_path"] = output_dir
                files.append(task)

            # 添加文件到会话
            if not self.service_client.add_files(files):
                self.service_client.end_session()
                self.logger.error("添加文件到服务会话失败，回退到传统模式")
                return self._process_files_batch_subprocess(file_list, output_dir, use_source_dir, naming_format)

            # 开始处理
            options = {
                "remove_source": False,
                "update_metadata": True,
                "overwrite_output": True,
                "skip_noop": True,
                "naming_format": naming_format
            }

            if not self.service_client.start_processing(options):
                self.service_client.end_session()
                self.logger.error("启动服务处理失败，回退到传统模式")
                return self._process_files_batch_subprocess(file_list, output_dir, use_source_dir, naming_format)

            # 等待处理完成并获取进度
            import time
            start_time = time.time()
            timeout = PROCESS_TIMEOUT_SECONDS * len(file_list)

            while True:
                progress_info = self.service_client.get_progress()
                if progress_info is None:
                    break

                status = progress_info.get("status", "unknown")
                if status in ["completed", "partial_success", "error"]:
                    break

                # 检查超时
                if time.time() - start_time > timeout:
                    self.service_client.stop_processing()
                    self.service_client.end_session()
                    return {
                        "success": False,
                        "error": f"服务处理超时（超过{timeout}秒）",
                        "results": []
                    }

                time.sleep(0.1)  # 等待100ms再检查（减少延迟）

            # 获取最终结果
            final_progress = self.service_client.get_progress()
            self.service_client.end_session()

            if final_progress:
                total_files = final_progress.get("total_files", len(file_list))
                processed_files = final_progress.get("processed_files", 0)
                status = final_progress.get("status", "unknown")

                success = status in ["completed", "partial_success"]
                return {
                    "success": success,
                    "success_count": processed_files,
                    "failed_count": total_files - processed_files,
                    "total_files": total_files,
                    "results": [],  # 服务模式暂不返回详细结果
                    "total_time_ms": int((time.time() - start_time) * 1000)
                }
            else:
                return {
                    "success": False,
                    "error": "无法获取处理结果",
                    "results": []
                }

        except Exception as e:
            self.logger.error(f"服务模式处理异常: {e}，回退到传统模式")
            if self.service_client and self.service_client.session_id:
                self.service_client.end_session()
            return self._process_files_batch_subprocess(file_list, output_dir, use_source_dir, naming_format)

    def _process_files_batch_subprocess(self, file_list: list, output_dir: str = None,
                                      use_source_dir: bool = False, naming_format: str = "auto") -> dict:
        """
        使用传统subprocess模式批量处理文件
        """
        import json

        try:
            # 构建批处理请求
            batch_request = {
                "files": [],
                "options": {
                    "remove_source": False,
                    "update_metadata": True,
                    "overwrite_output": True,
                    "skip_noop": True,
                    "naming_format": naming_format
                }
            }

            # 添加文件任务
            for file_path in file_list:
                task = {"input_path": file_path}

                if not use_source_dir and output_dir:
                    task["output_path"] = output_dir

                batch_request["files"].append(task)

            self.logger.info(f"开始传统模式批处理 {len(file_list)} 个文件")

            # 调用批处理模式
            cmd = [self.um_exe_path, "--batch"]

            result = subprocess.run(
                cmd,
                input=json.dumps(batch_request),
                capture_output=True,
                text=True,
                encoding='utf-8',
                errors='ignore',
                timeout=PROCESS_TIMEOUT_SECONDS * len(file_list),  # 根据文件数量调整超时
                **self._get_subprocess_kwargs()
            )

            if result.returncode == 0:
                # 解析批处理响应
                response = json.loads(result.stdout)
                self.logger.info(f"批处理完成: 成功 {response.get('success_count', 0)}, "
                               f"失败 {response.get('failed_count', 0)}, "
                               f"耗时 {response.get('total_time_ms', 0)}ms")
                return response
            else:
                error_msg = result.stderr or result.stdout or '未知错误'
                self.logger.error(f"批处理失败: {error_msg}")
                return {
                    "success": False,
                    "error": error_msg,
                    "results": []
                }

        except subprocess.TimeoutExpired:
            error_msg = f"批处理超时（超过{PROCESS_TIMEOUT_SECONDS * len(file_list)}秒）"
            self.logger.error(error_msg)
            return {
                "success": False,
                "error": error_msg,
                "results": []
            }
        except Exception as e:
            error_msg = f"批处理异常: {str(e)}"
            self.logger.error(error_msg)
            return {
                "success": False,
                "error": error_msg,
                "results": []
            }
