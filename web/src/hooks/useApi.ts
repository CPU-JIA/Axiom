import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'react-hot-toast';
import { 
  AuthAPI, 
  ProjectAPI, 
  TaskAPI, 
  TenantAPI, 
  DashboardAPI,
  LoginRequest,
  CreateProjectRequest,
  UpdateProjectRequest,
  CreateTaskRequest,
  UpdateTaskRequest,
  ProjectListParams
} from '../services/api';

// 查询键常量
export const QUERY_KEYS = {
  // 认证相关
  currentUser: ['auth', 'currentUser'] as const,
  
  // 项目相关
  projects: (params?: ProjectListParams) => ['projects', params] as const,
  project: (id: string) => ['projects', id] as const,
  
  // 任务相关
  tasks: (projectId?: string) => ['tasks', { projectId }] as const,
  task: (id: string) => ['tasks', id] as const,
  
  // 租户相关
  currentTenant: ['tenants', 'current'] as const,
  tenantMembers: ['tenants', 'current', 'members'] as const,
  
  // 仪表盘相关
  dashboardStats: ['dashboard', 'stats'] as const,
  recentActivities: (limit?: number) => ['dashboard', 'activities', { limit }] as const,
} as const;

// ====== 认证相关 Hooks ======
export const useLogin = () => {
  return useMutation({
    mutationFn: (data: LoginRequest) => AuthAPI.login(data),
    onSuccess: (response) => {
      toast.success('登录成功！');
      console.log('Login successful:', response.data);
    },
    onError: (error: any) => {
      console.error('Login failed:', error);
      toast.error(error.response?.data?.message || '登录失败，请重试');
    },
  });
};

export const useLogout = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: () => AuthAPI.logout(),
    onSuccess: () => {
      // 清除所有查询缓存
      queryClient.clear();
      toast.success('退出登录成功');
    },
    onError: (error: any) => {
      console.error('Logout failed:', error);
      toast.error('退出登录失败');
    },
  });
};

export const useCurrentUser = () => {
  return useQuery({
    queryKey: QUERY_KEYS.currentUser,
    queryFn: () => AuthAPI.getCurrentUser(),
    select: (response) => response.data,
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 1,
  });
};

// ====== 项目相关 Hooks ======
export const useProjects = (params?: ProjectListParams) => {
  return useQuery({
    queryKey: QUERY_KEYS.projects(params),
    queryFn: () => ProjectAPI.getProjects(params),
    select: (response) => response.data,
    staleTime: 2 * 60 * 1000, // 2 minutes
  });
};

export const useProject = (id: string) => {
  return useQuery({
    queryKey: QUERY_KEYS.project(id),
    queryFn: () => ProjectAPI.getProject(id),
    select: (response) => response.data,
    enabled: !!id,
  });
};

export const useCreateProject = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreateProjectRequest) => ProjectAPI.createProject(data),
    onSuccess: () => {
      // 刷新项目列表
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      // 刷新仪表盘统计
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.dashboardStats });
      
      toast.success('项目创建成功！');
    },
    onError: (error: any) => {
      console.error('Create project failed:', error);
      toast.error(error.response?.data?.message || '创建项目失败');
    },
  });
};

export const useUpdateProject = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateProjectRequest }) => 
      ProjectAPI.updateProject(id, data),
    onSuccess: (_, variables) => {
      // 更新特定项目缓存
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.project(variables.id) });
      // 刷新项目列表
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      // 刷新仪表盘统计
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.dashboardStats });
      
      toast.success('项目更新成功！');
    },
    onError: (error: any) => {
      console.error('Update project failed:', error);
      toast.error(error.response?.data?.message || '更新项目失败');
    },
  });
};

export const useDeleteProject = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => ProjectAPI.deleteProject(id),
    onSuccess: (_, deletedId) => {
      // 移除特定项目缓存
      queryClient.removeQueries({ queryKey: QUERY_KEYS.project(deletedId) });
      // 刷新项目列表
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      // 刷新仪表盘统计
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.dashboardStats });
      
      toast.success('项目删除成功！');
    },
    onError: (error: any) => {
      console.error('Delete project failed:', error);
      toast.error(error.response?.data?.message || '删除项目失败');
    },
  });
};

export const useToggleStarProject = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => ProjectAPI.toggleStarProject(id),
    onSuccess: (response, projectId) => {
      // 更新特定项目缓存
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.project(projectId) });
      // 刷新项目列表
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      
      const isStarred = response.data.isStarred;
      toast.success(isStarred ? '已添加到收藏' : '已取消收藏');
    },
    onError: (error: any) => {
      console.error('Toggle star failed:', error);
      toast.error(error.response?.data?.message || '操作失败');
    },
  });
};

// ====== 任务相关 Hooks ======
export const useTasks = (projectId?: string) => {
  return useQuery({
    queryKey: QUERY_KEYS.tasks(projectId),
    queryFn: () => TaskAPI.getTasks(projectId),
    select: (response) => response.data,
    staleTime: 1 * 60 * 1000, // 1 minute
  });
};

export const useTask = (id: string) => {
  return useQuery({
    queryKey: QUERY_KEYS.task(id),
    queryFn: () => TaskAPI.getTask(id),
    select: (response) => response.data,
    enabled: !!id,
  });
};

export const useCreateTask = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreateTaskRequest) => TaskAPI.createTask(data),
    onSuccess: (_, variables) => {
      // 刷新任务列表
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      // 更新特定项目的任务
      queryClient.invalidateQueries({ 
        queryKey: QUERY_KEYS.tasks(variables.projectId) 
      });
      // 刷新项目详情（更新任务统计）
      queryClient.invalidateQueries({ 
        queryKey: QUERY_KEYS.project(variables.projectId) 
      });
      // 刷新仪表盘统计
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.dashboardStats });
      
      toast.success('任务创建成功！');
    },
    onError: (error: any) => {
      console.error('Create task failed:', error);
      toast.error(error.response?.data?.message || '创建任务失败');
    },
  });
};

export const useUpdateTask = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateTaskRequest }) => 
      TaskAPI.updateTask(id, data),
    onSuccess: (_, variables) => {
      // 更新特定任务缓存
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.task(variables.id) });
      // 刷新任务列表
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      // 刷新项目相关缓存
      if (variables.data.projectId) {
        queryClient.invalidateQueries({ 
          queryKey: QUERY_KEYS.project(variables.data.projectId) 
        });
      }
      // 刷新仪表盘统计
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.dashboardStats });
      
      toast.success('任务更新成功！');
    },
    onError: (error: any) => {
      console.error('Update task failed:', error);
      toast.error(error.response?.data?.message || '更新任务失败');
    },
  });
};

export const useDeleteTask = () => {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: string) => TaskAPI.deleteTask(id),
    onSuccess: (_, deletedId) => {
      // 移除特定任务缓存
      queryClient.removeQueries({ queryKey: QUERY_KEYS.task(deletedId) });
      // 刷新任务列表
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      // 刷新仪表盘统计
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.dashboardStats });
      
      toast.success('任务删除成功！');
    },
    onError: (error: any) => {
      console.error('Delete task failed:', error);
      toast.error(error.response?.data?.message || '删除任务失败');
    },
  });
};

// ====== 租户相关 Hooks ======
export const useCurrentTenant = () => {
  return useQuery({
    queryKey: QUERY_KEYS.currentTenant,
    queryFn: () => TenantAPI.getCurrentTenant(),
    select: (response) => response.data,
    staleTime: 10 * 60 * 1000, // 10 minutes
  });
};

export const useTenantMembers = () => {
  return useQuery({
    queryKey: QUERY_KEYS.tenantMembers,
    queryFn: () => TenantAPI.getTenantMembers(),
    select: (response) => response.data,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
};

// ====== 仪表盘相关 Hooks ======
export const useDashboardStats = () => {
  return useQuery({
    queryKey: QUERY_KEYS.dashboardStats,
    queryFn: () => DashboardAPI.getStats(),
    select: (response) => response.data,
    staleTime: 1 * 60 * 1000, // 1 minute
    refetchInterval: 5 * 60 * 1000, // 每5分钟自动刷新
  });
};

export const useRecentActivities = (limit?: number) => {
  return useQuery({
    queryKey: QUERY_KEYS.recentActivities(limit),
    queryFn: () => DashboardAPI.getRecentActivities(limit),
    select: (response) => response.data,
    staleTime: 30 * 1000, // 30 seconds
    refetchInterval: 2 * 60 * 1000, // 每2分钟自动刷新
  });
};

// ====== 通用工具 Hooks ======

// 预加载数据
export const usePrefetchProject = () => {
  const queryClient = useQueryClient();
  
  return (id: string) => {
    queryClient.prefetchQuery({
      queryKey: QUERY_KEYS.project(id),
      queryFn: () => ProjectAPI.getProject(id),
      staleTime: 1 * 60 * 1000,
    });
  };
};

// 乐观更新工具
export const useOptimisticUpdate = () => {
  const queryClient = useQueryClient();
  
  return {
    updateProject: (id: string, updates: Partial<any>) => {
      queryClient.setQueryData(QUERY_KEYS.project(id), (old: any) => ({
        ...old,
        ...updates,
      }));
    },
    
    updateTask: (id: string, updates: Partial<any>) => {
      queryClient.setQueryData(QUERY_KEYS.task(id), (old: any) => ({
        ...old,
        ...updates,
      }));
    },
  };
};