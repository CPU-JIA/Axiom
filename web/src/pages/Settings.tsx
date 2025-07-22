import React, { useState } from 'react';
import { 
  User, 
  Shield, 
  Bell, 
  Palette, 
  Monitor,
  Smartphone,
  Mail,
  Phone,
  MapPin,
  Camera,
  Key,
  Eye,
  EyeOff,
  Save,
  X
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { Card, Button, Input, PageHeader, animations } from '../components/ui';
import { useNotifications } from '../components/NotificationSystem';
import { cn } from '../utils/cn';

interface UserProfile {
  id: string;
  name: string;
  email: string;
  phone?: string;
  avatar: string;
  title: string;
  department: string;
  location: string;
  bio: string;
  timezone: string;
  language: string;
}

interface NotificationSettings {
  email: {
    projectUpdates: boolean;
    taskAssignments: boolean;
    mentions: boolean;
    systemUpdates: boolean;
  };
  push: {
    projectUpdates: boolean;
    taskAssignments: boolean;
    mentions: boolean;
    systemUpdates: boolean;
  };
  desktop: {
    enabled: boolean;
    sound: boolean;
  };
}

interface SecuritySettings {
  twoFactorEnabled: boolean;
  loginAlerts: boolean;
  sessionTimeout: number;
  trustedDevices: string[];
}

interface AppearanceSettings {
  theme: 'light' | 'dark' | 'system';
  primaryColor: string;
  fontSize: 'small' | 'medium' | 'large';
  sidebarCollapsed: boolean;
}

const Settings: React.FC = () => {
  const { success, error } = useNotifications();
  const [activeTab, setActiveTab] = useState<'profile' | 'security' | 'notifications' | 'appearance'>('profile');
  const [isLoading, setIsLoading] = useState(false);

  // 用户资料状态
  const [profile, setProfile] = useState<UserProfile>({
    id: '1',
    name: 'JIA',
    email: 'jia@euclid-elements.com',
    phone: '+86 138 0013 8000',
    avatar: 'https://ui-avatars.com/api/?name=JIA&background=0ea5e9&color=fff',
    title: '首席技术官',
    department: '技术部',
    location: '北京，中国',
    bio: '专注于云计算和分布式系统架构设计，致力于打造企业级开发协作平台。',
    timezone: 'Asia/Shanghai',
    language: 'zh-CN'
  });

  // 通知设置状态
  const [notifications, setNotifications] = useState<NotificationSettings>({
    email: {
      projectUpdates: true,
      taskAssignments: true,
      mentions: true,
      systemUpdates: false
    },
    push: {
      projectUpdates: true,
      taskAssignments: true,
      mentions: true,
      systemUpdates: false
    },
    desktop: {
      enabled: true,
      sound: true
    }
  });

  // 安全设置状态
  const [security, setSecurity] = useState<SecuritySettings>({
    twoFactorEnabled: true,
    loginAlerts: true,
    sessionTimeout: 30,
    trustedDevices: ['iPhone 15 Pro', 'MacBook Pro', 'Chrome on Windows']
  });

  // 外观设置状态
  const [appearance, setAppearance] = useState<AppearanceSettings>({
    theme: 'system',
    primaryColor: '#0ea5e9',
    fontSize: 'medium',
    sidebarCollapsed: false
  });

  // 密码修改状态
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
    showCurrent: false,
    showNew: false,
    showConfirm: false
  });

  // 标签页配置
  const tabs = [
    { id: 'profile', label: '个人资料', icon: User },
    { id: 'security', label: '安全设置', icon: Shield },
    { id: 'notifications', label: '通知设置', icon: Bell },
    { id: 'appearance', label: '外观设置', icon: Palette }
  ];

  // 保存设置
  const handleSave = async () => {
    setIsLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      success('设置保存成功', '您的设置已经保存并生效');
    } catch (err) {
      error('保存失败', '请检查网络连接后重试');
    } finally {
      setIsLoading(false);
    }
  };

  // 头像上传
  const handleAvatarUpload = () => {
    // 模拟文件选择器
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = 'image/*';
    input.onchange = () => {
      success('头像上传成功', '您的头像已更新');
    };
    input.click();
  };

  // 个人资料标签页
  const ProfileTab = () => (
    <motion.div
      initial="initial"
      animate="animate"
      variants={animations.staggerContainer}
      className="space-y-6"
    >
      {/* 头像部分 */}
      <Card className="p-6">
        <motion.div variants={animations.fadeInUp} className="flex items-center space-x-6">
          <div className="relative">
            <img
              src={profile.avatar}
              alt={profile.name}
              className="w-24 h-24 rounded-2xl object-cover"
            />
            <button
              onClick={handleAvatarUpload}
              className="absolute -bottom-2 -right-2 bg-primary-600 text-white rounded-full p-2 hover:bg-primary-700 transition-colors shadow-lg"
            >
              <Camera className="w-4 h-4" />
            </button>
          </div>
          <div>
            <h3 className="text-xl font-semibold text-gray-900">{profile.name}</h3>
            <p className="text-gray-600">{profile.title}</p>
            <p className="text-sm text-gray-500 mt-1">{profile.department}</p>
          </div>
        </motion.div>
      </Card>

      {/* 基本信息 */}
      <Card className="p-6">
        <motion.div variants={animations.fadeInUp}>
          <h3 className="text-lg font-semibold text-gray-900 mb-4">基本信息</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Input
              label="姓名"
              value={profile.name}
              onChange={(e) => setProfile({ ...profile, name: e.target.value })}
              leftIcon={<User className="w-4 h-4" />}
            />
            <Input
              label="邮箱"
              type="email"
              value={profile.email}
              onChange={(e) => setProfile({ ...profile, email: e.target.value })}
              leftIcon={<Mail className="w-4 h-4" />}
            />
            <Input
              label="手机号"
              value={profile.phone || ''}
              onChange={(e) => setProfile({ ...profile, phone: e.target.value })}
              leftIcon={<Phone className="w-4 h-4" />}
            />
            <Input
              label="位置"
              value={profile.location}
              onChange={(e) => setProfile({ ...profile, location: e.target.value })}
              leftIcon={<MapPin className="w-4 h-4" />}
            />
          </div>
        </motion.div>
      </Card>

      {/* 个人简介 */}
      <Card className="p-6">
        <motion.div variants={animations.fadeInUp}>
          <h3 className="text-lg font-semibold text-gray-900 mb-4">个人简介</h3>
          <textarea
            value={profile.bio}
            onChange={(e) => setProfile({ ...profile, bio: e.target.value })}
            rows={4}
            className="w-full px-3 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-primary-500 resize-none"
            placeholder="介绍一下自己..."
          />
        </motion.div>
      </Card>
    </motion.div>
  );

  // 安全设置标签页
  const SecurityTab = () => (
    <motion.div
      initial="initial"
      animate="animate"
      variants={animations.staggerContainer}
      className="space-y-6"
    >
      {/* 密码修改 */}
      <Card className="p-6">
        <motion.div variants={animations.fadeInUp}>
          <h3 className="text-lg font-semibold text-gray-900 mb-4">修改密码</h3>
          <div className="space-y-4 max-w-md">
            <Input
              label="当前密码"
              type={passwordForm.showCurrent ? 'text' : 'password'}
              value={passwordForm.currentPassword}
              onChange={(e) => setPasswordForm({ ...passwordForm, currentPassword: e.target.value })}
              rightIcon={
                <button
                  type="button"
                  onClick={() => setPasswordForm({ ...passwordForm, showCurrent: !passwordForm.showCurrent })}
                >
                  {passwordForm.showCurrent ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              }
            />
            <Input
              label="新密码"
              type={passwordForm.showNew ? 'text' : 'password'}
              value={passwordForm.newPassword}
              onChange={(e) => setPasswordForm({ ...passwordForm, newPassword: e.target.value })}
              rightIcon={
                <button
                  type="button"
                  onClick={() => setPasswordForm({ ...passwordForm, showNew: !passwordForm.showNew })}
                >
                  {passwordForm.showNew ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              }
            />
            <Input
              label="确认新密码"
              type={passwordForm.showConfirm ? 'text' : 'password'}
              value={passwordForm.confirmPassword}
              onChange={(e) => setPasswordForm({ ...passwordForm, confirmPassword: e.target.value })}
              rightIcon={
                <button
                  type="button"
                  onClick={() => setPasswordForm({ ...passwordForm, showConfirm: !passwordForm.showConfirm })}
                >
                  {passwordForm.showConfirm ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              }
            />
            <Button leftIcon={<Key />}>更新密码</Button>
          </div>
        </motion.div>
      </Card>

      {/* 两步验证 */}
      <Card className="p-6">
        <motion.div variants={animations.fadeInUp}>
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-lg font-semibold text-gray-900">两步验证</h3>
              <p className="text-sm text-gray-600">为您的账户提供额外的安全保护</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={security.twoFactorEnabled}
                onChange={(e) => setSecurity({ ...security, twoFactorEnabled: e.target.checked })}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>
        </motion.div>
      </Card>

      {/* 受信任设备 */}
      <Card className="p-6">
        <motion.div variants={animations.fadeInUp}>
          <h3 className="text-lg font-semibold text-gray-900 mb-4">受信任设备</h3>
          <div className="space-y-3">
            {security.trustedDevices.map((device, index) => (
              <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-xl">
                <div className="flex items-center space-x-3">
                  <div className="p-2 bg-primary-100 rounded-lg">
                    {device.includes('iPhone') || device.includes('iPad') ? (
                      <Smartphone className="w-4 h-4 text-primary-600" />
                    ) : (
                      <Monitor className="w-4 h-4 text-primary-600" />
                    )}
                  </div>
                  <div>
                    <p className="font-medium text-gray-900">{device}</p>
                    <p className="text-sm text-gray-500">最后活动：2小时前</p>
                  </div>
                </div>
                <Button size="sm" variant="outline" leftIcon={<X />}>
                  移除
                </Button>
              </div>
            ))}
          </div>
        </motion.div>
      </Card>
    </motion.div>
  );

  // 通知设置标签页
  const NotificationsTab = () => (
    <motion.div
      initial="initial"
      animate="animate"
      variants={animations.staggerContainer}
      className="space-y-6"
    >
      {['email', 'push'].map((type) => (
        <Card key={type} className="p-6">
          <motion.div variants={animations.fadeInUp}>
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              {type === 'email' ? '邮件通知' : '推送通知'}
            </h3>
            <div className="space-y-4">
              {Object.entries(notifications[type as keyof typeof notifications] as any).map(([key, value]) => (
                <div key={key} className="flex items-center justify-between">
                  <div>
                    <p className="font-medium text-gray-900">
                      {key === 'projectUpdates' && '项目更新'}
                      {key === 'taskAssignments' && '任务分配'}
                      {key === 'mentions' && '提及通知'}
                      {key === 'systemUpdates' && '系统更新'}
                    </p>
                    <p className="text-sm text-gray-500">
                      {key === 'projectUpdates' && '项目状态变更时通知'}
                      {key === 'taskAssignments' && '被分配新任务时通知'}
                      {key === 'mentions' && '在评论中被提及时通知'}
                      {key === 'systemUpdates' && '系统维护和更新时通知'}
                    </p>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      checked={value as boolean}
                      onChange={(e) => setNotifications({
                        ...notifications,
                        [type]: {
                          ...notifications[type as keyof typeof notifications],
                          [key]: e.target.checked
                        }
                      })}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
                  </label>
                </div>
              ))}
            </div>
          </motion.div>
        </Card>
      ))}
    </motion.div>
  );

  // 外观设置标签页
  const AppearanceTab = () => (
    <motion.div
      initial="initial"
      animate="animate"
      variants={animations.staggerContainer}
      className="space-y-6"
    >
      {/* 主题设置 */}
      <Card className="p-6">
        <motion.div variants={animations.fadeInUp}>
          <h3 className="text-lg font-semibold text-gray-900 mb-4">主题设置</h3>
          <div className="grid grid-cols-3 gap-4">
            {[
              { id: 'light', label: '浅色模式', desc: '经典的浅色界面' },
              { id: 'dark', label: '深色模式', desc: '护眼的深色界面' },
              { id: 'system', label: '跟随系统', desc: '根据系统设置自动切换' }
            ].map((theme) => (
              <button
                key={theme.id}
                onClick={() => setAppearance({ ...appearance, theme: theme.id as any })}
                className={cn(
                  'p-4 border-2 rounded-xl text-left transition-colors',
                  appearance.theme === theme.id
                    ? 'border-primary-500 bg-primary-50'
                    : 'border-gray-200 hover:border-gray-300'
                )}
              >
                <h4 className="font-medium text-gray-900">{theme.label}</h4>
                <p className="text-sm text-gray-600 mt-1">{theme.desc}</p>
              </button>
            ))}
          </div>
        </motion.div>
      </Card>

      {/* 字体大小 */}
      <Card className="p-6">
        <motion.div variants={animations.fadeInUp}>
          <h3 className="text-lg font-semibold text-gray-900 mb-4">字体大小</h3>
          <div className="flex space-x-4">
            {[
              { id: 'small', label: '小', size: 'text-sm' },
              { id: 'medium', label: '中', size: 'text-base' },
              { id: 'large', label: '大', size: 'text-lg' }
            ].map((size) => (
              <button
                key={size.id}
                onClick={() => setAppearance({ ...appearance, fontSize: size.id as any })}
                className={cn(
                  'px-4 py-2 rounded-xl border-2 transition-colors',
                  size.size,
                  appearance.fontSize === size.id
                    ? 'border-primary-500 bg-primary-50 text-primary-700'
                    : 'border-gray-200 hover:border-gray-300'
                )}
              >
                {size.label}字体
              </button>
            ))}
          </div>
        </motion.div>
      </Card>
    </motion.div>
  );

  const renderTabContent = () => {
    switch (activeTab) {
      case 'profile':
        return <ProfileTab />;
      case 'security':
        return <SecurityTab />;
      case 'notifications':
        return <NotificationsTab />;
      case 'appearance':
        return <AppearanceTab />;
      default:
        return <ProfileTab />;
    }
  };

  return (
    <div className="flex-1 p-6 max-w-7xl mx-auto">
      <PageHeader
        title="设置"
        description="管理您的账户设置和个性化偏好"
        actions={
          <Button
            loading={isLoading}
            onClick={handleSave}
            leftIcon={<Save />}
          >
            保存设置
          </Button>
        }
      />

      <div className="flex flex-col lg:flex-row gap-6">
        {/* 侧边栏导航 */}
        <motion.div
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          className="lg:w-64"
        >
          <Card className="p-2">
            <nav className="space-y-1">
              {tabs.map((tab) => {
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id as any)}
                    className={cn(
                      'w-full flex items-center px-3 py-2 text-sm font-medium rounded-xl transition-colors',
                      activeTab === tab.id
                        ? 'bg-primary-100 text-primary-700'
                        : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                    )}
                  >
                    <Icon className="w-4 h-4 mr-3" />
                    {tab.label}
                  </button>
                );
              })}
            </nav>
          </Card>
        </motion.div>

        {/* 主内容区域 */}
        <motion.div
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          className="flex-1"
        >
          <AnimatePresence mode="wait">
            <motion.div
              key={activeTab}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.2 }}
            >
              {renderTabContent()}
            </motion.div>
          </AnimatePresence>
        </motion.div>
      </div>
    </div>
  );
};

export default Settings;