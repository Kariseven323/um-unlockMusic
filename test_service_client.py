#!/usr/bin/env python3
"""
测试CLI服务模式的简单客户端
"""

import json
import socket
import time
import uuid
import platform
import os

class UMServiceClient:
    def __init__(self, pipe_name=None):
        self.pipe_name = pipe_name or self._get_default_pipe()
        self.socket = None
        self.is_windows = platform.system() == 'Windows'
        
    def _get_default_pipe(self):
        """获取默认管道名称"""
        if platform.system() == 'Windows':
            return r'\\.\pipe\um_service'
        else:
            return '/tmp/um_service.sock'

    def connect(self):
        """连接到服务"""
        try:
            if self.is_windows:
                # Windows命名管道 - 使用ctypes调用Windows API
                import ctypes
                from ctypes import wintypes

                # Windows API常量
                GENERIC_READ = 0x80000000
                GENERIC_WRITE = 0x40000000
                OPEN_EXISTING = 3
                INVALID_HANDLE_VALUE = -1

                # 调用CreateFile API
                kernel32 = ctypes.windll.kernel32
                self.socket = kernel32.CreateFileW(
                    self.pipe_name,
                    GENERIC_READ | GENERIC_WRITE,
                    0,
                    None,
                    OPEN_EXISTING,
                    0,
                    None
                )

                if self.socket == INVALID_HANDLE_VALUE:
                    raise Exception(f"无法打开命名管道: {ctypes.GetLastError()}")

                print(f"✅ 成功连接到Windows命名管道 {self.pipe_name}")
            else:
                # Unix域套接字
                self.socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
                self.socket.connect(self.pipe_name)
                print(f"✅ 成功连接到Unix套接字 {self.pipe_name}")
            return True
        except Exception as e:
            print(f"❌ 连接失败: {e}")
            return False
    
    def disconnect(self):
        """断开连接"""
        if self.socket:
            if self.is_windows:
                import ctypes
                kernel32 = ctypes.windll.kernel32
                kernel32.CloseHandle(self.socket)
            else:
                self.socket.close()
            self.socket = None
            print("🔌 已断开连接")
    
    def send_message(self, msg_type, data=None):
        """发送消息到服务"""
        if not self.socket:
            print("❌ 未连接到服务")
            return None
            
        message = {
            "id": str(uuid.uuid4()),
            "type": msg_type,
            "data": data or {},
            "timestamp": int(time.time())
        }
        
        try:
            # 发送JSON消息
            json_str = json.dumps(message, ensure_ascii=False)
            message_bytes = (json_str + '\n').encode('utf-8')

            if self.is_windows:
                import win32file
                win32file.WriteFile(self.socket, message_bytes)
            else:
                self.socket.send(message_bytes)
            print(f"📤 发送消息: {msg_type}")

            # 接收响应
            if self.is_windows:
                import win32file
                result, response_data = win32file.ReadFile(self.socket, 4096)
            else:
                response_data = self.socket.recv(4096)

            if response_data:
                response_str = response_data.decode('utf-8').strip()
                response = json.loads(response_str)
                print(f"📥 收到响应: {response.get('type', 'unknown')}")
                return response
            else:
                print("❌ 未收到响应")
                return None
                
        except Exception as e:
            print(f"❌ 通信错误: {e}")
            return None

def test_service_communication():
    """测试服务通信"""
    print("🚀 开始测试CLI服务模式通信")
    print("=" * 50)
    
    client = UMServiceClient()
    
    # 连接到服务
    if not client.connect():
        return
    
    try:
        # 测试1: 启动会话
        print("\n📋 测试1: 启动会话")
        response = client.send_message("start_session")
        if response and response.get('success'):
            session_id = response.get('data', {}).get('session_id')
            print(f"✅ 会话启动成功，ID: {session_id}")
        else:
            print(f"❌ 会话启动失败: {response}")
        
        # 测试2: 添加文件
        print("\n📋 测试2: 添加文件")
        response = client.send_message("add_files", {
            "session_id": session_id,
            "files": ["test1.mflac", "test2.mflac"]
        })
        if response and response.get('success'):
            print("✅ 文件添加成功")
        else:
            print(f"❌ 文件添加失败: {response}")
        
        # 测试3: 获取进度
        print("\n📋 测试3: 获取进度")
        response = client.send_message("get_progress", {
            "session_id": session_id
        })
        if response and response.get('success'):
            progress_data = response.get('data', {})
            print(f"✅ 进度查询成功: {progress_data}")
        else:
            print(f"❌ 进度查询失败: {response}")
        
        # 测试4: 结束会话
        print("\n📋 测试4: 结束会话")
        response = client.send_message("end_session", {
            "session_id": session_id
        })
        if response and response.get('success'):
            print("✅ 会话结束成功")
        else:
            print(f"❌ 会话结束失败: {response}")
            
    except KeyboardInterrupt:
        print("\n⚠️ 用户中断测试")
    except Exception as e:
        print(f"\n❌ 测试过程中出错: {e}")
    finally:
        client.disconnect()
    
    print("\n" + "=" * 50)
    print("🏁 测试完成")

if __name__ == "__main__":
    test_service_communication()
