// 项目管理相关类型定义
export interface Project {
  id: string;
  name: string;
  description: string;
  status: ProjectStatus;
  priority: ProjectPriority;
  startDate: string;
  endDate?: string;
  estimatedHours?: number;
  actualHours?: number;
  progress: number;
  owner: ProjectMember;
  members: ProjectMember[];
  tags: string[];
  repository?: {
    url: string;
    branch: string;
    lastCommit?: string;
  };
  createdAt: string;
  updatedAt: string;
}

export interface ProjectMember {
  id: string;
  name: string;
  email: string;
  avatar?: string;
  role: ProjectRole;
  joinedAt: string;
}

export enum ProjectStatus {
  PLANNING = 'planning',
  IN_PROGRESS = 'in_progress',
  ON_HOLD = 'on_hold',
  COMPLETED = 'completed',
  CANCELLED = 'cancelled'
}

export enum ProjectPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  URGENT = 'urgent'
}

export enum ProjectRole {
  OWNER = 'owner',
  LEAD = 'lead',
  DEVELOPER = 'developer',
  TESTER = 'tester',
  OBSERVER = 'observer'
}

// 任务相关类型
export interface Task {
  id: string;
  title: string;
  description?: string;
  status: TaskStatus;
  priority: TaskPriority;
  type: TaskType;
  assignee: ProjectMember | null;
  reporter: ProjectMember;
  projectId: string;
  sprintId?: string;
  parentTaskId?: string;
  subtasks: Task[];
  estimatedHours?: number;
  actualHours?: number;
  storyPoints?: number;
  tags: string[];
  attachments: TaskAttachment[];
  comments: TaskComment[];
  dueDate?: string;
  startDate?: string;
  completedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export enum TaskStatus {
  BACKLOG = 'backlog',
  TODO = 'todo',
  IN_PROGRESS = 'in_progress',
  IN_REVIEW = 'in_review',
  TESTING = 'testing',
  DONE = 'done',
  BLOCKED = 'blocked'
}

export enum TaskPriority {
  LOWEST = 'lowest',
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  HIGHEST = 'highest'
}

export enum TaskType {
  FEATURE = 'feature',
  BUG = 'bug',
  IMPROVEMENT = 'improvement',
  DOCUMENTATION = 'documentation',
  RESEARCH = 'research',
  REFACTOR = 'refactor'
}

export interface TaskAttachment {
  id: string;
  name: string;
  url: string;
  size: number;
  mimeType: string;
  uploadedBy: string;
  uploadedAt: string;
}

export interface TaskComment {
  id: string;
  content: string;
  author: ProjectMember;
  createdAt: string;
  updatedAt?: string;
  isEdited: boolean;
}

// Sprint 相关类型
export interface Sprint {
  id: string;
  name: string;
  goal?: string;
  projectId: string;
  status: SprintStatus;
  startDate: string;
  endDate: string;
  tasks: Task[];
  totalStoryPoints: number;
  completedStoryPoints: number;
  createdAt: string;
  updatedAt: string;
}

export enum SprintStatus {
  PLANNED = 'planned',
  ACTIVE = 'active',
  COMPLETED = 'completed',
  CANCELLED = 'cancelled'
}

// 看板相关类型
export interface Board {
  id: string;
  name: string;
  projectId: string;
  columns: BoardColumn[];
  createdAt: string;
  updatedAt: string;
}

export interface BoardColumn {
  id: string;
  name: string;
  status: TaskStatus;
  order: number;
  limit?: number;
  tasks: Task[];
}

// 项目统计类型
export interface ProjectStats {
  totalTasks: number;
  completedTasks: number;
  inProgressTasks: number;
  overdueTasksCount: number;
  totalMembers: number;
  totalHours: number;
  completionRate: number;
  velocity: number;
  burndownData: BurndownPoint[];
}

export interface BurndownPoint {
  date: string;
  remaining: number;
  ideal: number;
}