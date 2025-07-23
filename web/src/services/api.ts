import axios from 'axios';
import { useAuthStore } from '../stores/authStore';

// API 基础配置
const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

// 创建axios实例
const apiClient = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
apiClient.interceptors.request.use(
  (config: any) => {
    // 获取token并添加到请求头
    const token = useAuthStore.getState().token;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    
    // 添加请求时间戳
    config.metadata = { startTime: new Date() };
    
    console.log(`🚀 API Request: ${config.method?.toUpperCase()} ${config.url}`);
    return config;
  },
  (error: any) => {
    console.error('❌ Request Error:', error);
    return Promise.reject(error);
  }
);

// 响应拦截器
apiClient.interceptors.response.use(
  (response: any) => {
    const duration = new Date().getTime() - response.config.metadata?.startTime?.getTime();
    console.log(`✅ API Response: ${response.config.method?.toUpperCase()} ${response.config.url} (${duration}ms)`);
    
    return response;
  },
  (error: any) => {
    const duration = error.config?.metadata?.startTime ? 
      new Date().getTime() - error.config.metadata.startTime.getTime() : 0;
    
    console.error(`❌ API Error: ${error.config?.method?.toUpperCase()} ${error.config?.url} (${duration}ms)`, {
      status: error.response?.status,
      statusText: error.response?.statusText,
      data: error.response?.data
    });

    // 处理401未授权错误
    if (error.response?.status === 401) {
      useAuthStore.getState().logout();
      window.location.href = '/login';
    }

    return Promise.reject(error);
  }
);

// API响应类型
interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}

// API服务类
class ApiService {
  // GET请求
  static async get<T = any>(url: string, config?: any): Promise<ApiResponse<T>> {
    const response = await apiClient.get(url, config);
    return response.data;
  }

  // POST请求
  static async post<T = any>(url: string, data?: any, config?: any): Promise<ApiResponse<T>> {
    const response = await apiClient.post(url, data, config);
    return response.data;
  }

  // PUT请求
  static async put<T = any>(url: string, data?: any, config?: any): Promise<ApiResponse<T>> {
    const response = await apiClient.put(url, data, config);
    return response.data;
  }

  // PATCH请求
  static async patch<T = any>(url: string, data?: any, config?: any): Promise<ApiResponse<T>> {
    const response = await apiClient.patch(url, data, config);
    return response.data;
  }

  // DELETE请求
  static async delete<T = any>(url: string, config?: any): Promise<ApiResponse<T>> {
    const response = await apiClient.delete(url, config);
    return response.data;
  }
}

// 用户相关API
export const userApi = {
  // 获取当前用户信息
  getCurrentUser: () => ApiService.get('/user/profile'),
  
  // 更新用户信息
  updateProfile: (data: any) => ApiService.put('/user/profile', data),
  
  // 获取用户列表
  getUsers: (params?: any) => ApiService.get('/users', { params }),
};

// 认证相关API
export const authApi = {
  // 登录
  login: (credentials: { email: string; password: string }) => 
    ApiService.post('/auth/login', credentials),
  
  // 注册
  register: (userData: any) => ApiService.post('/auth/register', userData),
  
  // 刷新token
  refreshToken: () => ApiService.post('/auth/refresh'),
  
  // 登出
  logout: () => ApiService.post('/auth/logout'),
  
  // 忘记密码
  forgotPassword: (email: string) => ApiService.post('/auth/forgot-password', { email }),
  
  // 重置密码
  resetPassword: (token: string, password: string) => 
    ApiService.post('/auth/reset-password', { token, password }),
};

// 项目相关API
export const projectApi = {
  // 获取项目列表
  getProjects: (params?: any) => ApiService.get('/projects', { params }),
  
  // 获取项目详情
  getProject: (id: string) => ApiService.get(`/projects/${id}`),
  
  // 创建项目
  createProject: (data: any) => ApiService.post('/projects', data),
  
  // 更新项目
  updateProject: (id: string, data: any) => ApiService.put(`/projects/${id}`, data),
  
  // 删除项目
  deleteProject: (id: string) => ApiService.delete(`/projects/${id}`),
  
  // 获取项目成员
  getProjectMembers: (id: string) => ApiService.get(`/projects/${id}/members`),
  
  // 添加项目成员
  addProjectMember: (id: string, data: any) => ApiService.post(`/projects/${id}/members`, data),
  
  // 移除项目成员
  removeProjectMember: (id: string, memberId: string) => 
    ApiService.delete(`/projects/${id}/members/${memberId}`),
};

// 任务相关API
export const taskApi = {
  // 获取任务列表
  getTasks: (params?: any) => ApiService.get('/tasks', { params }),
  
  // 获取任务详情
  getTask: (id: string) => ApiService.get(`/tasks/${id}`),
  
  // 创建任务
  createTask: (data: any) => ApiService.post('/tasks', data),
  
  // 更新任务
  updateTask: (id: string, data: any) => ApiService.put(`/tasks/${id}`, data),
  
  // 删除任务
  deleteTask: (id: string) => ApiService.delete(`/tasks/${id}`),
  
  // 获取任务评论
  getTaskComments: (id: string) => ApiService.get(`/tasks/${id}/comments`),
  
  // 添加任务评论
  addTaskComment: (id: string, data: any) => ApiService.post(`/tasks/${id}/comments`, data),
  
  // 上传任务附件
  uploadTaskAttachment: (id: string, file: FormData) => 
    ApiService.post(`/tasks/${id}/attachments`, file, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }),
};

// 仪表板相关API
export const dashboardApi = {
  // 获取仪表板数据
  getDashboardData: () => ApiService.get('/dashboard'),
  
  // 获取活动日志
  getActivities: (params?: any) => ApiService.get('/dashboard/activities', { params }),
  
  // 获取统计数据
  getStats: () => ApiService.get('/dashboard/stats'),
};

// 导出默认API服务
export default ApiService;