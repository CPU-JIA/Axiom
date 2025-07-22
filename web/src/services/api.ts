import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { useAuthStore } from '../stores/authStore';

// API 基础配置
const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

// 创建axios实例
const apiClient: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
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
  (error) => {
    console.error('❌ Request Error:', error);
    return Promise.reject(error);
  }
);

// 响应拦截器
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    const duration = new Date().getTime() - response.config.metadata?.startTime?.getTime();
    console.log(`✅ API Response: ${response.config.method?.toUpperCase()} ${response.config.url} (${duration}ms)`);
    
    return response;
  },
  (error) => {
    const duration = error.config?.metadata?.startTime ? 
      new Date().getTime() - error.config.metadata.startTime.getTime() : 0;
    
    console.error(`❌ API Error: ${error.config?.method?.toUpperCase()} ${error.config?.url} (${duration}ms)`, error.response?.data);
    
    // 处理认证错误
    if (error.response?.status === 401) {
      // 清除认证状态并重定向到登录页
      useAuthStore.getState().logout();
      window.location.href = '/login';
    }
    
    // 处理其他HTTP错误
    if (error.response?.status >= 500) {
      // 服务器错误，可以显示用户友好的错误信息
      console.error('Server error occurred');
    }
    
    return Promise.reject(error);
  }
);

// API响应类型定义
export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
  timestamp: string;
}

export interface PaginationResponse<T = any> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// 通用API方法
export class ApiService {
  static async get<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await apiClient.get(url, config);
    return response.data;
  }

  static async post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await apiClient.post(url, data, config);
    return response.data;
  }

  static async put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await apiClient.put(url, data, config);
    return response.data;
  }

  static async patch<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await apiClient.patch(url, data, config);
    return response.data;
  }

  static async delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await apiClient.delete(url, config);
    return response.data;
  }
}

// 用户相关API
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: {
    id: string;
    email: string;
    name: string;
    avatar?: string;
    role: string;
    tenantId: string;
  };
  token: string;
  refreshToken: string;
  expiresIn: number;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatar?: string;
  role: string;
  tenantId: string;
  createdAt: string;
  updatedAt: string;
}

export class AuthAPI {
  static async login(data: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return ApiService.post('/auth/login', data);
  }

  static async logout(): Promise<ApiResponse<void>> {
    return ApiService.post('/auth/logout');
  }

  static async refresh(refreshToken: string): Promise<ApiResponse<LoginResponse>> {
    return ApiService.post('/auth/refresh', { refreshToken });
  }

  static async getCurrentUser(): Promise<ApiResponse<User>> {
    return ApiService.get('/auth/me');
  }
}

// 项目相关API
export interface Project {
  id: string;
  name: string;
  description: string;
  status: 'planning' | 'active' | 'testing' | 'completed' | 'paused';
  priority: 'low' | 'medium' | 'high' | 'critical';
  progress: number;
  startDate: string;
  endDate: string;
  members: ProjectMember[];
  tasksTotal: number;
  tasksCompleted: number;
  isStarred: boolean;
  lastActivity: string;
  repository?: string;
  tenantId: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectMember {
  id: string;
  name: string;
  email: string;
  avatar: string;
  role: string;
}

export interface CreateProjectRequest {
  name: string;
  description: string;
  priority: Project['priority'];
  startDate: string;
  endDate: string;
  memberIds: string[];
}

export interface UpdateProjectRequest extends Partial<CreateProjectRequest> {
  status?: Project['status'];
  progress?: number;
}

export interface ProjectListParams {
  page?: number;
  pageSize?: number;
  status?: string;
  priority?: string;
  search?: string;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export class ProjectAPI {
  static async getProjects(params?: ProjectListParams): Promise<ApiResponse<PaginationResponse<Project>>> {
    return ApiService.get('/projects', { params });
  }

  static async getProject(id: string): Promise<ApiResponse<Project>> {
    return ApiService.get(`/projects/${id}`);
  }

  static async createProject(data: CreateProjectRequest): Promise<ApiResponse<Project>> {
    return ApiService.post('/projects', data);
  }

  static async updateProject(id: string, data: UpdateProjectRequest): Promise<ApiResponse<Project>> {
    return ApiService.put(`/projects/${id}`, data);
  }

  static async deleteProject(id: string): Promise<ApiResponse<void>> {
    return ApiService.delete(`/projects/${id}`);
  }

  static async toggleStarProject(id: string): Promise<ApiResponse<Project>> {
    return ApiService.post(`/projects/${id}/star`);
  }
}

// 任务相关API
export interface Task {
  id: string;
  title: string;
  description: string;
  status: 'todo' | 'in_progress' | 'in_review' | 'done';
  priority: 'low' | 'medium' | 'high' | 'critical';
  assigneeId?: string;
  assignee?: ProjectMember;
  projectId: string;
  dueDate?: string;
  tags: string[];
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateTaskRequest {
  title: string;
  description: string;
  priority: Task['priority'];
  projectId: string;
  assigneeId?: string;
  dueDate?: string;
  tags: string[];
}

export interface UpdateTaskRequest extends Partial<CreateTaskRequest> {
  status?: Task['status'];
}

export class TaskAPI {
  static async getTasks(projectId?: string): Promise<ApiResponse<Task[]>> {
    const params = projectId ? { projectId } : {};
    return ApiService.get('/tasks', { params });
  }

  static async getTask(id: string): Promise<ApiResponse<Task>> {
    return ApiService.get(`/tasks/${id}`);
  }

  static async createTask(data: CreateTaskRequest): Promise<ApiResponse<Task>> {
    return ApiService.post('/tasks', data);
  }

  static async updateTask(id: string, data: UpdateTaskRequest): Promise<ApiResponse<Task>> {
    return ApiService.put(`/tasks/${id}`, data);
  }

  static async deleteTask(id: string): Promise<ApiResponse<void>> {
    return ApiService.delete(`/tasks/${id}`);
  }
}

// 租户相关API
export interface Tenant {
  id: string;
  name: string;
  domain: string;
  logo?: string;
  settings: Record<string, any>;
  subscriptionPlan: string;
  subscriptionStatus: 'active' | 'inactive' | 'trial' | 'expired';
  memberCount: number;
  createdAt: string;
  updatedAt: string;
}

export class TenantAPI {
  static async getCurrentTenant(): Promise<ApiResponse<Tenant>> {
    return ApiService.get('/tenants/current');
  }

  static async updateTenant(data: Partial<Tenant>): Promise<ApiResponse<Tenant>> {
    return ApiService.put('/tenants/current', data);
  }

  static async getTenantMembers(): Promise<ApiResponse<User[]>> {
    return ApiService.get('/tenants/current/members');
  }
}

// 统计数据API
export interface DashboardStats {
  totalProjects: number;
  activeProjects: number;
  completedTasks: number;
  totalTasks: number;
  teamMembers: number;
  completionRate: number;
}

export interface ActivityItem {
  id: string;
  user: string;
  action: string;
  target: string;
  time: string;
  type: 'commit' | 'task' | 'deployment' | 'comment';
}

export class DashboardAPI {
  static async getStats(): Promise<ApiResponse<DashboardStats>> {
    return ApiService.get('/dashboard/stats');
  }

  static async getRecentActivities(limit?: number): Promise<ApiResponse<ActivityItem[]>> {
    const params = limit ? { limit } : {};
    return ApiService.get('/dashboard/activities', { params });
  }
}

// 导出默认的API客户端
export default apiClient;