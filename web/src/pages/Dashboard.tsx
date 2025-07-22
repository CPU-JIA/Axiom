import React, { useState } from 'react';
import { 
  Home, 
  FolderKanban, 
  Users, 
  GitBranch, 
  Settings, 
  Bell, 
  Search, 
  Plus,
  BarChart3,
  Calendar,
  Clock,
  TrendingUp,
  Activity,
  CheckCircle,
  AlertCircle,
  Menu,
  X,
  Sparkles,
  ChevronDown
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

interface Project {
  id: string;
  name: string;
  status: 'active' | 'completed' | 'paused';
  progress: number;
  members: number;
  dueDate: string;
}

interface Activity {
  id: string;
  user: string;
  action: string;
  target: string;
  time: string;
  type: 'commit' | 'task' | 'deployment' | 'comment';
}

const Dashboard: React.FC = () => {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [activeTab, setActiveTab] = useState('overview');

  // 模拟数据
  const projects: Project[] = [
    { id: '1', name: '用户管理系统', status: 'active', progress: 75, members: 5, dueDate: '2024-02-15' },
    { id: '2', name: 'API网关服务', status: 'active', progress: 60, members: 3, dueDate: '2024-02-20' },
    { id: '3', name: 'CI/CD管道', status: 'completed', progress: 100, members: 4, dueDate: '2024-01-30' },
    { id: '4', name: '监控告警系统', status: 'paused', progress: 30, members: 2, dueDate: '2024-03-01' }
  ];

  const activities: Activity[] = [
    { id: '1', user: 'JIA', action: '提交了代码到', target: '用户管理系统', time: '2分钟前', type: 'commit' },
    { id: '2', user: '张三', action: '完成了任务', target: 'JWT认证模块', time: '15分钟前', type: 'task' },
    { id: '3', user: '李四', action: '部署了', target: 'API网关服务 v1.2.0', time: '1小时前', type: 'deployment' },
    { id: '4', user: '王五', action: '评论了', target: '数据库设计文档', time: '2小时前', type: 'comment' },
  ];

  const stats = [
    { label: '总项目数', value: '12', change: '+2', trend: 'up' },
    { label: '进行中任务', value: '34', change: '+5', trend: 'up' },
    { label: '团队成员', value: '8', change: '0', trend: 'stable' },
    { label: '完成率', value: '87%', change: '+3%', trend: 'up' }
  ];

  const menuItems = [
    { id: 'overview', label: '概览', icon: Home, active: true },
    { id: 'projects', label: '项目管理', icon: FolderKanban, active: false },
    { id: 'team', label: '团队协作', icon: Users, active: false },
    { id: 'git', label: 'Git管理', icon: GitBranch, active: false },
    { id: 'analytics', label: '数据分析', icon: BarChart3, active: false },
    { id: 'settings', label: '系统设置', icon: Settings, active: false },
  ];

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'text-green-600 bg-green-100';
      case 'completed': return 'text-blue-600 bg-blue-100';
      case 'paused': return 'text-yellow-600 bg-yellow-100';
      default: return 'text-gray-600 bg-gray-100';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'active': return '进行中';
      case 'completed': return '已完成';
      case 'paused': return '已暂停';
      default: return '未知';
    }
  };

  const getActivityIcon = (type: string) => {
    switch (type) {
      case 'commit': return <GitBranch className="w-4 h-4 text-blue-500" />;
      case 'task': return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'deployment': return <Activity className="w-4 h-4 text-purple-500" />;
      case 'comment': return <AlertCircle className="w-4 h-4 text-orange-500" />;
      default: return <Clock className="w-4 h-4 text-gray-500" />;
    }
  };

  return (
    <div className="flex h-screen bg-gray-50">
      {/* 侧边栏 */}
      <AnimatePresence>
        {sidebarOpen && (
          <motion.div
            initial={{ x: -300, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            exit={{ x: -300, opacity: 0 }}
            transition={{ duration: 0.3 }}
            className="fixed inset-y-0 left-0 z-50 w-64 bg-white shadow-strong lg:static lg:inset-0"
          >
            <div className="flex items-center justify-between h-16 px-6 border-b border-gray-200">
              <div className="flex items-center">
                <div className="flex items-center justify-center w-8 h-8 bg-gradient-to-br from-primary-500 to-primary-600 rounded-lg mr-3">
                  <Sparkles className="w-5 h-5 text-white" />
                </div>
                <span className="text-xl font-bold text-gray-900">几何原本</span>
              </div>
              <button
                onClick={() => setSidebarOpen(false)}
                className="lg:hidden text-gray-400 hover:text-gray-600"
              >
                <X className="w-6 h-6" />
              </button>
            </div>

            <nav className="mt-8 px-4">
              <div className="space-y-2">
                {menuItems.map((item) => {
                  const Icon = item.icon;
                  const isActive = activeTab === item.id;
                  
                  return (
                    <motion.button
                      key={item.id}
                      whileHover={{ scale: 1.02 }}
                      whileTap={{ scale: 0.98 }}
                      onClick={() => setActiveTab(item.id)}
                      className={`w-full flex items-center px-4 py-3 text-left rounded-xl transition-all duration-200 ${
                        isActive
                          ? 'bg-primary-50 text-primary-700 border-l-4 border-primary-500'
                          : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                      }`}
                    >
                      <Icon className={`w-5 h-5 mr-3 ${isActive ? 'text-primary-500' : ''}`} />
                      {item.label}
                    </motion.button>
                  );
                })}
              </div>
            </nav>

            {/* 底部用户信息 */}
            <div className="absolute bottom-0 left-0 right-0 p-4">
              <div className="flex items-center p-3 bg-gray-50 rounded-xl">
                <div className="w-10 h-10 bg-gradient-to-br from-primary-500 to-primary-600 rounded-full flex items-center justify-center text-white font-semibold">
                  J
                </div>
                <div className="ml-3 flex-1">
                  <p className="text-sm font-medium text-gray-900">JIA</p>
                  <p className="text-xs text-gray-500">系统管理员</p>
                </div>
                <ChevronDown className="w-4 h-4 text-gray-400" />
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* 主内容区域 */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* 顶部导航栏 */}
        <header className="bg-white shadow-sm border-b border-gray-200">
          <div className="flex items-center justify-between px-6 py-4">
            <div className="flex items-center">
              <button
                onClick={() => setSidebarOpen(!sidebarOpen)}
                className="text-gray-500 hover:text-gray-700 lg:hidden"
              >
                <Menu className="w-6 h-6" />
              </button>
              <h1 className="text-2xl font-bold text-gray-900 ml-4 lg:ml-0">工作台概览</h1>
            </div>

            <div className="flex items-center space-x-4">
              {/* 搜索框 */}
              <div className="relative hidden md:block">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <Search className="h-5 w-5 text-gray-400" />
                </div>
                <input
                  type="text"
                  placeholder="搜索项目、任务..."
                  className="block w-64 pl-10 pr-3 py-2 border border-gray-300 rounded-lg leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              {/* 通知按钮 */}
              <button className="relative p-2 text-gray-400 hover:text-gray-500 focus:outline-none">
                <Bell className="w-6 h-6" />
                <span className="absolute top-0 right-0 block h-2 w-2 rounded-full bg-red-400"></span>
              </button>

              {/* 新建按钮 */}
              <motion.button
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm text-white bg-gradient-to-r from-primary-600 to-primary-700 hover:from-primary-700 hover:to-primary-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
              >
                <Plus className="w-4 h-4 mr-2" />
                新建项目
              </motion.button>
            </div>
          </div>
        </header>

        {/* 主内容 */}
        <main className="flex-1 overflow-y-auto p-6">
          {/* 统计卡片 */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            {stats.map((stat, index) => (
              <motion.div
                key={stat.label}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.1 }}
                className="bg-white rounded-2xl shadow-soft p-6 hover:shadow-medium transition-shadow duration-200"
              >
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-600">{stat.label}</p>
                    <p className="text-3xl font-bold text-gray-900 mt-2">{stat.value}</p>
                  </div>
                  <div className={`flex items-center text-sm ${
                    stat.trend === 'up' ? 'text-green-600' : 
                    stat.trend === 'down' ? 'text-red-600' : 'text-gray-500'
                  }`}>
                    {stat.trend === 'up' && <TrendingUp className="w-4 h-4 mr-1" />}
                    {stat.change}
                  </div>
                </div>
              </motion.div>
            ))}
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* 项目概览 */}
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.2 }}
              className="bg-white rounded-2xl shadow-soft p-6"
            >
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-semibold text-gray-900">项目概览</h2>
                <a href="#" className="text-primary-600 hover:text-primary-500 text-sm font-medium">
                  查看全部
                </a>
              </div>

              <div className="space-y-4">
                {projects.map((project) => (
                  <div key={project.id} className="border border-gray-200 rounded-xl p-4 hover:bg-gray-50 transition-colors">
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="font-medium text-gray-900">{project.name}</h3>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(project.status)}`}>
                        {getStatusText(project.status)}
                      </span>
                    </div>
                    
                    <div className="flex items-center justify-between text-sm text-gray-500 mb-3">
                      <span>{project.members} 成员</span>
                      <span>截止: {project.dueDate}</span>
                    </div>
                    
                    <div className="w-full bg-gray-200 rounded-full h-2">
                      <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${project.progress}%` }}
                        transition={{ duration: 1, delay: 0.5 }}
                        className="bg-gradient-to-r from-primary-500 to-primary-600 h-2 rounded-full"
                      ></motion.div>
                    </div>
                    <div className="text-right text-xs text-gray-500 mt-1">{project.progress}%</div>
                  </div>
                ))}
              </div>
            </motion.div>

            {/* 活动时间线 */}
            <motion.div
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.3 }}
              className="bg-white rounded-2xl shadow-soft p-6"
            >
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-semibold text-gray-900">最近活动</h2>
                <a href="#" className="text-primary-600 hover:text-primary-500 text-sm font-medium">
                  查看全部
                </a>
              </div>

              <div className="space-y-4">
                {activities.map((activity) => (
                  <div key={activity.id} className="flex items-start space-x-3">
                    <div className="flex-shrink-0 w-8 h-8 bg-gray-100 rounded-full flex items-center justify-center">
                      {getActivityIcon(activity.type)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-gray-900">
                        <span className="font-medium">{activity.user}</span>{' '}
                        {activity.action}{' '}
                        <span className="font-medium text-primary-600">{activity.target}</span>
                      </p>
                      <p className="text-xs text-gray-500 mt-1">{activity.time}</p>
                    </div>
                  </div>
                ))}
              </div>
            </motion.div>
          </div>
        </main>
      </div>

      {/* 移动端遮罩 */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black bg-opacity-25 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}
    </div>
  );
};

export default Dashboard;