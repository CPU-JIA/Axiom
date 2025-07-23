import React, { useState, useMemo } from 'react';
import { DragDropContext, Droppable, Draggable } from '@hello-pangea/dnd';
import { 
  Plus, 
  Search, 
  Filter, 
  Calendar, 
  Clock, 
  MessageCircle,
  Paperclip,
  MoreHorizontal,
  Eye
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { Card, Button, Input, PageHeader } from '../components/ui';
import { animations } from '../components/ui';
import { cn } from '../utils/cn';
import { formatRelativeTime } from '../utils/cn';

// 任务数据类型定义
interface Task {
  id: string;
  title: string;
  description?: string;
  priority: 'low' | 'medium' | 'high' | 'critical';
  status: 'todo' | 'in_progress' | 'in_review' | 'done';
  assignee?: {
    id: string;
    name: string;
    avatar: string;
  };
  dueDate?: string;
  tags: string[];
  attachments: number;
  comments: number;
  subtasks?: { completed: number; total: number };
  createdAt: string;
}

interface Column {
  id: string;
  title: string;
  status: Task['status'];
  color: string;
  limit?: number;
}

const TaskBoard: React.FC = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedPriority, setSelectedPriority] = useState<string>('all');
  const [selectedAssignee, setSelectedAssignee] = useState<string>('all');

  // 看板列配置
  const columns: Column[] = [
    { id: 'todo', title: '待办事项', status: 'todo', color: 'bg-gray-100', limit: 10 },
    { id: 'in_progress', title: '进行中', status: 'in_progress', color: 'bg-blue-100', limit: 5 },
    { id: 'in_review', title: '待审核', status: 'in_review', color: 'bg-yellow-100', limit: 3 },
    { id: 'done', title: '已完成', status: 'done', color: 'bg-green-100' }
  ];

  // 模拟任务数据
  const [tasks, setTasks] = useState<Task[]>([
    {
      id: '1',
      title: '用户认证系统重构',
      description: '重构现有的用户认证系统，支持多因子认证',
      priority: 'high',
      status: 'in_progress',
      assignee: {
        id: '1',
        name: 'JIA',
        avatar: 'https://ui-avatars.com/api/?name=JIA&background=0ea5e9&color=fff'
      },
      dueDate: '2024-02-15',
      tags: ['后端', '安全'],
      attachments: 3,
      comments: 8,
      subtasks: { completed: 3, total: 5 },
      createdAt: '2024-01-20T10:00:00Z'
    },
    {
      id: '2',
      title: 'API文档更新',
      description: '更新所有API端点的文档说明',
      priority: 'medium',
      status: 'todo',
      assignee: {
        id: '2',
        name: '张三',
        avatar: 'https://ui-avatars.com/api/?name=张三&background=10b981&color=fff'
      },
      dueDate: '2024-02-20',
      tags: ['文档'],
      attachments: 1,
      comments: 2,
      createdAt: '2024-01-22T14:30:00Z'
    },
    {
      id: '3',
      title: '前端性能优化',
      description: '优化首页加载时间，提升用户体验',
      priority: 'critical',
      status: 'in_review',
      assignee: {
        id: '3',
        name: '李四',
        avatar: 'https://ui-avatars.com/api/?name=李四&background=f59e0b&color=fff'
      },
      dueDate: '2024-02-10',
      tags: ['前端', '性能'],
      attachments: 0,
      comments: 5,
      subtasks: { completed: 8, total: 8 },
      createdAt: '2024-01-15T09:15:00Z'
    },
    {
      id: '4',
      title: '数据库备份策略',
      description: '制定和实施数据库定期备份策略',
      priority: 'high',
      status: 'done',
      assignee: {
        id: '4',
        name: '王五',
        avatar: 'https://ui-avatars.com/api/?name=王五&background=8b5cf6&color=fff'
      },
      tags: ['数据库', 'DevOps'],
      attachments: 2,
      comments: 12,
      createdAt: '2024-01-10T16:45:00Z'
    },
    {
      id: '5',
      title: '移动端适配',
      description: '确保所有页面在移动设备上正常显示',
      priority: 'medium',
      status: 'todo',
      tags: ['前端', '移动端'],
      attachments: 0,
      comments: 1,
      createdAt: '2024-01-25T11:20:00Z'
    }
  ]);

  // 优先级配置
  const priorityConfig = {
    critical: { label: '紧急', color: 'text-red-600 bg-red-100', icon: '🔥' },
    high: { label: '高', color: 'text-orange-600 bg-orange-100', icon: '⚡' },
    medium: { label: '中', color: 'text-yellow-600 bg-yellow-100', icon: '📋' },
    low: { label: '低', color: 'text-green-600 bg-green-100', icon: '🌱' }
  };

  // 过滤任务
  const filteredTasks = useMemo(() => {
    return tasks.filter(task => {
      const matchesSearch = task.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           task.description?.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesPriority = selectedPriority === 'all' || task.priority === selectedPriority;
      const matchesAssignee = selectedAssignee === 'all' || task.assignee?.id === selectedAssignee;
      
      return matchesSearch && matchesPriority && matchesAssignee;
    });
  }, [tasks, searchTerm, selectedPriority, selectedAssignee]);

  // 按状态分组任务
  const tasksByStatus = useMemo(() => {
    const grouped = columns.reduce((acc, column) => {
      acc[column.status] = filteredTasks.filter(task => task.status === column.status);
      return acc;
    }, {} as Record<Task['status'], Task[]>);
    
    return grouped;
  }, [filteredTasks, columns]);

  // 拖拽处理
  const handleDragEnd = (result: any) => {
    const { destination, source, draggableId } = result;

    if (!destination) return;
    if (destination.droppableId === source.droppableId && destination.index === source.index) return;

    const newTasks = Array.from(tasks);
    const taskIndex = newTasks.findIndex(task => task.id === draggableId);
    
    if (taskIndex !== -1) {
      newTasks[taskIndex] = {
        ...newTasks[taskIndex],
        status: destination.droppableId as Task['status']
      };
      setTasks(newTasks);
    }
  };

  // 任务卡片组件
  const TaskCard: React.FC<{ task: Task; index: number }> = ({ task, index }) => {
    const isOverdue = task.dueDate && new Date(task.dueDate) < new Date();
    
    return (
      <Draggable draggableId={task.id} index={index}>
        {(provided, snapshot) => {
          // 分离的motion props
          const motionProps = {
            initial: { opacity: 0, y: 20 },
            animate: { opacity: 1, y: 0 },
            transition: { delay: index * 0.05 },
            className: cn(
              'group mb-3 transition-all duration-200',
              snapshot.isDragging && 'rotate-3 scale-105 shadow-2xl'
            )
          };
          
          return (
            <motion.div
              {...motionProps}
            >
              <div
                ref={provided.innerRef}
                {...provided.draggableProps}
                {...provided.dragHandleProps}
              >
            <Card className={cn(
              'p-4 hover:shadow-medium border-l-4 relative',
              task.priority === 'critical' && 'border-l-red-500',
              task.priority === 'high' && 'border-l-orange-500',
              task.priority === 'medium' && 'border-l-yellow-500',
              task.priority === 'low' && 'border-l-green-500',
              snapshot.isDragging && 'bg-white shadow-2xl'
            )}>
              {/* 任务头部 */}
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center space-x-2">
                  <span className="text-sm">
                    {priorityConfig[task.priority].icon}
                  </span>
                  <span className={cn(
                    'px-2 py-1 rounded-full text-xs font-medium',
                    priorityConfig[task.priority].color
                  )}>
                    {priorityConfig[task.priority].label}
                  </span>
                  {isOverdue && (
                    <span className="px-2 py-1 rounded-full text-xs font-medium text-red-600 bg-red-100">
                      已逾期
                    </span>
                  )}
                </div>
                <button className="opacity-0 group-hover:opacity-100 transition-opacity text-gray-400 hover:text-gray-600">
                  <MoreHorizontal className="w-4 h-4" />
                </button>
              </div>

              {/* 任务标题和描述 */}
              <h3 className="font-semibold text-gray-900 mb-2 line-clamp-2">{task.title}</h3>
              {task.description && (
                <p className="text-sm text-gray-600 mb-3 line-clamp-2">{task.description}</p>
              )}

              {/* 子任务进度 */}
              {task.subtasks && (
                <div className="mb-3">
                  <div className="flex items-center justify-between text-xs text-gray-500 mb-1">
                    <span>子任务进度</span>
                    <span>{task.subtasks.completed}/{task.subtasks.total}</span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <motion.div
                      initial={{ width: 0 }}
                      animate={{ width: `${(task.subtasks.completed / task.subtasks.total) * 100}%` }}
                      transition={{ duration: 0.5, delay: 0.2 }}
                      className="bg-primary-500 h-2 rounded-full"
                    />
                  </div>
                </div>
              )}

              {/* 标签 */}
              {task.tags.length > 0 && (
                <div className="flex flex-wrap gap-1 mb-3">
                  {task.tags.map((tag, tagIndex) => (
                    <span
                      key={tagIndex}
                      className="px-2 py-1 bg-gray-100 text-gray-600 text-xs rounded-md"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              )}

              {/* 任务底部信息 */}
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  {task.assignee && (
                    <div className="flex items-center">
                      <img
                        src={task.assignee.avatar}
                        alt={task.assignee.name}
                        className="w-6 h-6 rounded-full"
                      />
                    </div>
                  )}
                  
                  <div className="flex items-center space-x-2 text-gray-500">
                    {task.attachments > 0 && (
                      <div className="flex items-center text-xs">
                        <Paperclip className="w-3 h-3 mr-1" />
                        {task.attachments}
                      </div>
                    )}
                    
                    {task.comments > 0 && (
                      <div className="flex items-center text-xs">
                        <MessageCircle className="w-3 h-3 mr-1" />
                        {task.comments}
                      </div>
                    )}
                  </div>
                </div>

                {task.dueDate && (
                  <div className={cn(
                    'flex items-center text-xs',
                    isOverdue ? 'text-red-600' : 'text-gray-500'
                  )}>
                    <Clock className="w-3 h-3 mr-1" />
                    {formatRelativeTime(task.dueDate)}
                  </div>
                )}
              </div>
            </Card>
              </div>
            </motion.div>
          );
        }}
      </Draggable>
    );
  };

  // 列组件
  const Column: React.FC<{ column: Column }> = ({ column }) => {
    const columnTasks = tasksByStatus[column.status] || [];
    const isOverLimit = column.limit && columnTasks.length > column.limit;

    return (
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex-1 min-w-80"
      >
        <div className={cn(
          'rounded-2xl p-4 h-full',
          column.color,
          'min-h-[600px]'
        )}>
          {/* 列头 */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center space-x-2">
              <h2 className="font-semibold text-gray-900">{column.title}</h2>
              <span className={cn(
                'px-2 py-1 rounded-full text-xs font-medium bg-white',
                isOverLimit ? 'text-red-600' : 'text-gray-600'
              )}>
                {columnTasks.length}
                {column.limit && `/${column.limit}`}
              </span>
            </div>
            
            <Button size="sm" variant="ghost">
              <Plus className="w-4 h-4" />
            </Button>
          </div>

          {/* 任务列表 */}
          <Droppable droppableId={column.status}>
            {(provided, snapshot) => (
              <div
                ref={provided.innerRef}
                {...provided.droppableProps}
                className={cn(
                  'min-h-[500px] transition-colors duration-200 rounded-xl p-2',
                  snapshot.isDraggingOver && 'bg-white/50'
                )}
              >
                <AnimatePresence>
                  {columnTasks.map((task, index) => (
                    <TaskCard key={task.id} task={task} index={index} />
                  ))}
                </AnimatePresence>
                {provided.placeholder}
                
                {columnTasks.length === 0 && (
                  <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 0.5 }}
                    className="flex items-center justify-center h-32 text-gray-500 text-sm border-2 border-dashed border-gray-300 rounded-xl"
                  >
                    拖拽任务到此处
                  </motion.div>
                )}
              </div>
            )}
          </Droppable>
        </div>
      </motion.div>
    );
  };

  return (
    <div className="flex-1 p-6 bg-gray-50 min-h-screen">
      <PageHeader
        title="任务看板"
        description="拖拽管理您的任务进度，提升团队协作效率"
        breadcrumb={[
          { name: '工作台', href: '/dashboard' },
          { name: '任务看板' }
        ]}
        actions={
          <>
            <Button variant="outline" leftIcon={<Eye />}>
              视图设置
            </Button>
            <Button leftIcon={<Plus />}>
              新建任务
            </Button>
          </>
        }
      />

      {/* 筛选栏 */}
      <motion.div
        initial="initial"
        animate="animate"
        variants={animations.staggerContainer}
        className="bg-white rounded-2xl p-6 mb-6 shadow-soft"
      >
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <motion.div variants={animations.fadeInUp}>
            <Input
              placeholder="搜索任务..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              leftIcon={<Search className="w-4 h-4" />}
            />
          </motion.div>
          
          <motion.div variants={animations.fadeInUp}>
            <select
              value={selectedPriority}
              onChange={(e) => setSelectedPriority(e.target.value)}
              className="w-full px-3 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="all">所有优先级</option>
              <option value="critical">紧急</option>
              <option value="high">高</option>
              <option value="medium">中</option>
              <option value="low">低</option>
            </select>
          </motion.div>
          
          <motion.div variants={animations.fadeInUp}>
            <select
              value={selectedAssignee}
              onChange={(e) => setSelectedAssignee(e.target.value)}
              className="w-full px-3 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="all">所有成员</option>
              <option value="1">JIA</option>
              <option value="2">张三</option>
              <option value="3">李四</option>
              <option value="4">王五</option>
            </select>
          </motion.div>
          
          <motion.div variants={animations.fadeInUp} className="flex space-x-2">
            <Button variant="outline" leftIcon={<Filter />} className="flex-1">
              高级筛选
            </Button>
            <Button variant="outline" leftIcon={<Calendar />}>
              日期
            </Button>
          </motion.div>
        </div>
      </motion.div>

      {/* 看板主体 */}
      <DragDropContext onDragEnd={handleDragEnd}>
        <div className="flex space-x-6 overflow-x-auto pb-6">
          {columns.map((column) => (
            <Column key={column.id} column={column} />
          ))}
        </div>
      </DragDropContext>

      {/* 统计信息 */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.3 }}
        className="mt-8 grid grid-cols-1 md:grid-cols-4 gap-6"
      >
        {columns.map((column) => {
          const count = tasksByStatus[column.status]?.length || 0;
          return (
            <Card key={column.id} className="p-4 text-center">
              <div className="text-2xl font-bold text-gray-900 mb-1">{count}</div>
              <div className="text-sm text-gray-600">{column.title}</div>
              {column.limit && (
                <div className="text-xs text-gray-500 mt-1">
                  限制: {column.limit}
                </div>
              )}
            </Card>
          );
        })}
      </motion.div>
    </div>
  );
};

export default TaskBoard;