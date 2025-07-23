import React, { useState, useMemo } from 'react';
import { 
  Plus, 
  Search, 
  MoreHorizontal, 
  Calendar,
  GitBranch,
  Activity,
  CheckCircle,
  Edit,
  Eye,
  Star,
  StarOff,
  Folder
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

interface Project {
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
}

interface ProjectMember {
  id: string;
  name: string;
  avatar: string;
  role: string;
}

const ProjectManagement: React.FC = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [, setShowCreateModal] = useState(false);

  // 模拟项目数据
  const projects: Project[] = [
    {
      id: '1',
      name: '用户认证系统',
      description: '实现JWT认证、多因子认证和权限管理功能',
      status: 'active',
      priority: 'high',
      progress: 75,
      startDate: '2024-01-15',
      endDate: '2024-02-28',
      members: [
        { id: '1', name: 'JIA', avatar: 'https://ui-avatars.com/api/?name=JIA&background=0ea5e9&color=fff', role: 'Tech Lead' },
        { id: '2', name: '张三', avatar: 'https://ui-avatars.com/api/?name=张三&background=10b981&color=fff', role: 'Developer' },
        { id: '3', name: '李四', avatar: 'https://ui-avatars.com/api/?name=李四&background=f59e0b&color=fff', role: 'QA' }
      ],
      tasksTotal: 24,
      tasksCompleted: 18,
      isStarred: true,
      lastActivity: '2小时前',
      repository: 'euclid/auth-service'
    },
    {
      id: '2',
      name: 'API网关服务',
      description: '构建高性能的API网关，支持路由、限流、认证等功能',
      status: 'active',
      priority: 'medium',
      progress: 60,
      startDate: '2024-01-20',
      endDate: '2024-03-15',
      members: [
        { id: '4', name: '王五', avatar: 'https://ui-avatars.com/api/?name=王五&background=8b5cf6&color=fff', role: 'Developer' },
        { id: '5', name: '赵六', avatar: 'https://ui-avatars.com/api/?name=赵六&background=ec4899&color=fff', role: 'DevOps' }
      ],
      tasksTotal: 18,
      tasksCompleted: 11,
      isStarred: false,
      lastActivity: '1天前',
      repository: 'euclid/api-gateway'
    },
    {
      id: '3',
      name: '监控告警系统',
      description: '实现系统监控、日志收集和智能告警功能',
      status: 'planning',
      priority: 'medium',
      progress: 15,
      startDate: '2024-02-01',
      endDate: '2024-04-01',
      members: [
        { id: '6', name: '孙七', avatar: 'https://ui-avatars.com/api/?name=孙七&background=ef4444&color=fff', role: 'SRE' }
      ],
      tasksTotal: 32,
      tasksCompleted: 5,
      isStarred: false,
      lastActivity: '3天前'
    },
    {
      id: '4',
      name: 'CI/CD管道',
      description: '自动化构建、测试和部署流程',
      status: 'completed',
      priority: 'high',
      progress: 100,
      startDate: '2023-12-01',
      endDate: '2024-01-31',
      members: [
        { id: '1', name: 'JIA', avatar: 'https://ui-avatars.com/api/?name=JIA&background=0ea5e9&color=fff', role: 'Architect' },
        { id: '5', name: '赵六', avatar: 'https://ui-avatars.com/api/?name=赵六&background=ec4899&color=fff', role: 'DevOps' }
      ],
      tasksTotal: 15,
      tasksCompleted: 15,
      isStarred: true,
      lastActivity: '1周前',
      repository: 'euclid/cicd-pipeline'
    }
  ];

  // 过滤项目
  const filteredProjects = useMemo(() => {
    return projects.filter(project => {
      const matchesSearch = project.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           project.description.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesStatus = filterStatus === 'all' || project.status === filterStatus;
      return matchesSearch && matchesStatus;
    });
  }, [projects, searchTerm, filterStatus]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'planning': return 'bg-blue-100 text-blue-800';
      case 'active': return 'bg-green-100 text-green-800';
      case 'testing': return 'bg-yellow-100 text-yellow-800';
      case 'completed': return 'bg-gray-100 text-gray-800';
      case 'paused': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'planning': return '规划中';
      case 'active': return '进行中';
      case 'testing': return '测试中';
      case 'completed': return '已完成';
      case 'paused': return '已暂停';
      default: return '未知';
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'critical': return 'text-red-600';
      case 'high': return 'text-orange-600';
      case 'medium': return 'text-yellow-600';
      case 'low': return 'text-green-600';
      default: return 'text-gray-600';
    }
  };

  const getPriorityText = (priority: string) => {
    switch (priority) {
      case 'critical': return '紧急';
      case 'high': return '高';
      case 'medium': return '中';
      case 'low': return '低';
      default: return '未设置';
    }
  };

  return (
    <div className="flex-1 p-6">
      {/* 页面头部 */}
      <div className="mb-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">项目管理</h1>
            <p className="text-gray-600 mt-1">管理和跟踪所有项目的进展情况</p>
          </div>
          <motion.button
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
            onClick={() => setShowCreateModal(true)}
            className="mt-4 sm:mt-0 inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm text-white bg-gradient-to-r from-primary-600 to-primary-700 hover:from-primary-700 hover:to-primary-800"
          >
            <Plus className="w-4 h-4 mr-2" />
            新建项目
          </motion.button>
        </div>

        {/* 搜索和过滤栏 */}
        <div className="flex flex-col sm:flex-row gap-4 mb-6">
          <div className="flex-1">
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Search className="h-5 w-5 text-gray-400" />
              </div>
              <input
                type="text"
                placeholder="搜索项目名称或描述..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg leading-5 bg-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>

          <select
            value={filterStatus}
            onChange={(e) => setFilterStatus(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          >
            <option value="all">所有状态</option>
            <option value="planning">规划中</option>
            <option value="active">进行中</option>
            <option value="testing">测试中</option>
            <option value="completed">已完成</option>
            <option value="paused">已暂停</option>
          </select>

          <div className="flex rounded-lg border border-gray-300 overflow-hidden">
            <button
              onClick={() => setViewMode('grid')}
              className={`px-3 py-2 text-sm font-medium ${
                viewMode === 'grid' 
                  ? 'bg-primary-50 text-primary-700 border-primary-200' 
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              网格
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={`px-3 py-2 text-sm font-medium border-l border-gray-300 ${
                viewMode === 'list' 
                  ? 'bg-primary-50 text-primary-700 border-primary-200' 
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              列表
            </button>
          </div>
        </div>
      </div>

      {/* 项目列表 */}
      <div className={`${viewMode === 'grid' ? 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6' : 'space-y-4'}`}>
        <AnimatePresence>
          {filteredProjects.map((project) => (
            <motion.div
              key={project.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              className={`bg-white rounded-2xl shadow-soft hover:shadow-medium transition-all duration-200 ${
                viewMode === 'grid' ? 'p-6' : 'p-4 flex items-center space-x-4'
              }`}
            >
              {viewMode === 'grid' ? (
                // 网格视图
                <>
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <h3 className="text-lg font-semibold text-gray-900 truncate">{project.name}</h3>
                        <button
                          onClick={() => {}}
                          className="text-gray-400 hover:text-yellow-500 transition-colors"
                        >
                          {project.isStarred ? (
                            <Star className="w-4 h-4 fill-current text-yellow-500" />
                          ) : (
                            <StarOff className="w-4 h-4" />
                          )}
                        </button>
                      </div>
                      <p className="text-sm text-gray-600 line-clamp-2">{project.description}</p>
                    </div>
                    
                    <div className="flex items-center space-x-2 ml-4">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(project.status)}`}>
                        {getStatusText(project.status)}
                      </span>
                      <button className="text-gray-400 hover:text-gray-600">
                        <MoreHorizontal className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  <div className="space-y-3 mb-4">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-500">进度</span>
                      <span className="font-medium">{project.progress}%</span>
                    </div>
                    <div className="w-full bg-gray-200 rounded-full h-2">
                      <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${project.progress}%` }}
                        transition={{ duration: 1, delay: 0.3 }}
                        className="bg-gradient-to-r from-primary-500 to-primary-600 h-2 rounded-full"
                      />
                    </div>

                    <div className="flex items-center justify-between text-sm text-gray-500">
                      <div className="flex items-center">
                        <CheckCircle className="w-4 h-4 mr-1" />
                        {project.tasksCompleted}/{project.tasksTotal} 任务
                      </div>
                      <div className={`font-medium ${getPriorityColor(project.priority)}`}>
                        {getPriorityText(project.priority)}优先级
                      </div>
                    </div>

                    <div className="flex items-center justify-between text-sm text-gray-500">
                      <div className="flex items-center">
                        <Calendar className="w-4 h-4 mr-1" />
                        {project.endDate}
                      </div>
                      <div className="flex items-center">
                        <Activity className="w-4 h-4 mr-1" />
                        {project.lastActivity}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-4 border-t border-gray-100">
                    <div className="flex -space-x-2">
                      {project.members.slice(0, 3).map((member) => (
                        <img
                          key={member.id}
                          src={member.avatar}
                          alt={member.name}
                          className="w-8 h-8 rounded-full border-2 border-white"
                          title={member.name}
                        />
                      ))}
                      {project.members.length > 3 && (
                        <div className="w-8 h-8 rounded-full border-2 border-white bg-gray-100 flex items-center justify-center text-xs font-medium text-gray-600">
                          +{project.members.length - 3}
                        </div>
                      )}
                    </div>
                    
                    <div className="flex items-center space-x-2">
                      {project.repository && (
                        <div className="flex items-center text-xs text-gray-500">
                          <GitBranch className="w-3 h-3 mr-1" />
                          {project.repository}
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center space-x-2 mt-4">
                    <button className="flex-1 px-3 py-2 text-sm font-medium text-primary-700 bg-primary-50 rounded-lg hover:bg-primary-100 transition-colors">
                      <Eye className="w-4 h-4 mr-2 inline" />
                      查看详情
                    </button>
                    <button className="px-3 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors">
                      <Edit className="w-4 h-4" />
                    </button>
                  </div>
                </>
              ) : (
                // 列表视图
                <>
                  <div className="flex-1 grid grid-cols-1 md:grid-cols-6 gap-4 items-center">
                    <div className="md:col-span-2">
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-gray-900">{project.name}</h3>
                        {project.isStarred && (
                          <Star className="w-4 h-4 fill-current text-yellow-500" />
                        )}
                      </div>
                      <p className="text-sm text-gray-600 truncate">{project.description}</p>
                    </div>
                    
                    <div>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(project.status)}`}>
                        {getStatusText(project.status)}
                      </span>
                    </div>
                    
                    <div className="flex items-center">
                      <div className="w-16 bg-gray-200 rounded-full h-2 mr-2">
                        <div 
                          className="bg-gradient-to-r from-primary-500 to-primary-600 h-2 rounded-full"
                          style={{ width: `${project.progress}%` }}
                        />
                      </div>
                      <span className="text-sm font-medium">{project.progress}%</span>
                    </div>
                    
                    <div className="flex -space-x-2">
                      {project.members.slice(0, 3).map((member) => (
                        <img
                          key={member.id}
                          src={member.avatar}
                          alt={member.name}
                          className="w-6 h-6 rounded-full border-2 border-white"
                          title={member.name}
                        />
                      ))}
                    </div>
                    
                    <div className="text-sm text-gray-500">
                      {project.lastActivity}
                    </div>
                  </div>
                  
                  <button className="text-gray-400 hover:text-gray-600">
                    <MoreHorizontal className="w-5 h-5" />
                  </button>
                </>
              )}
            </motion.div>
          ))}
        </AnimatePresence>
      </div>

      {/* 空状态 */}
      {filteredProjects.length === 0 && (
        <div className="text-center py-12">
          <div className="w-24 h-24 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <Folder className="w-12 h-12 text-gray-400" />
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">没有找到项目</h3>
          <p className="text-gray-500 mb-6">
            {searchTerm || filterStatus !== 'all' ? '调整搜索条件或过滤器' : '开始创建您的第一个项目'}
          </p>
          {!searchTerm && filterStatus === 'all' && (
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => setShowCreateModal(true)}
              className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm text-white bg-primary-600 hover:bg-primary-700"
            >
              <Plus className="w-4 h-4 mr-2" />
              新建项目
            </motion.button>
          )}
        </div>
      )}
    </div>
  );
};

export default ProjectManagement;