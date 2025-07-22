import React, { useState } from 'react';
import { Eye, EyeOff, Lock, Mail, ArrowRight, Sparkles } from 'lucide-react';
import { motion } from 'framer-motion';
import { useNavigate } from 'react-router-dom';
import { useLogin } from '../hooks/useApi';
import { useAuthStore } from '../stores/authStore';

interface LoginFormData {
  email: string;
  password: string;
  rememberMe: boolean;
}

const Login: React.FC = () => {
  const navigate = useNavigate();
  const { setUser, setToken } = useAuthStore();
  const loginMutation = useLogin();

  const [formData, setFormData] = useState<LoginFormData>({
    email: '',
    password: '',
    rememberMe: false
  });
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<Partial<LoginFormData>>({});

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
    
    // 清除错误状态
    if (errors[name as keyof LoginFormData]) {
      setErrors(prev => ({ ...prev, [name]: '' }));
    }
  };

  const validateForm = (): boolean => {
    const newErrors: Partial<LoginFormData> = {};
    
    if (!formData.email) {
      newErrors.email = '请输入邮箱地址';
    } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
      newErrors.email = '请输入有效的邮箱地址';
    }
    
    if (!formData.password) {
      newErrors.password = '请输入密码';
    } else if (formData.password.length < 6) {
      newErrors.password = '密码长度至少6位';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) return;
    
    try {
      const response = await loginMutation.mutateAsync({
        email: formData.email,
        password: formData.password
      });
      
      // 更新认证状态
      setUser(response.data.user);
      setToken(response.data.token);
      
      // 导航到仪表盘
      navigate('/dashboard', { replace: true });
    } catch (error) {
      // 错误已经在mutation中处理了
      console.error('Login failed:', error);
    }
  };

  return (
    <div className="min-h-screen flex">
      {/* 左侧品牌展示区域 */}
      <div className="hidden lg:flex lg:flex-1 relative overflow-hidden gradient-bg">
        {/* 背景装饰元素 */}
        <div className="absolute inset-0">
          <div className="absolute top-20 left-20 w-32 h-32 bg-white/10 rounded-full blur-xl floating-animation"></div>
          <div className="absolute bottom-32 right-32 w-48 h-48 bg-white/5 rounded-full blur-2xl floating-animation" style={{ animationDelay: '2s' }}></div>
          <div className="absolute top-1/2 left-1/3 w-24 h-24 bg-white/15 rounded-full blur-lg floating-animation" style={{ animationDelay: '4s' }}></div>
        </div>
        
        {/* 内容区域 */}
        <div className="relative z-10 flex flex-col justify-center items-center text-white px-12">
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 1 }}
            className="text-center"
          >
            <div className="mb-8">
              <div className="inline-flex items-center justify-center w-16 h-16 bg-white/20 rounded-2xl backdrop-blur-sm mb-6">
                <Sparkles className="w-8 h-8 text-white" />
              </div>
              <h1 className="text-4xl font-bold mb-4">几何原本</h1>
              <p className="text-xl text-white/80">Euclid Elements</p>
            </div>
            
            <div className="space-y-4 text-left max-w-md">
              <blockquote className="text-lg text-white/90 italic leading-relaxed">
                "我们不创造天才，我们只是为天才们，构建一个配得上他们智慧的宇宙。"
              </blockquote>
              
              <div className="pt-8 space-y-3">
                <div className="flex items-center text-white/80">
                  <div className="w-2 h-2 bg-white/60 rounded-full mr-3"></div>
                  开发者心流至上
                </div>
                <div className="flex items-center text-white/80">
                  <div className="w-2 h-2 bg-white/60 rounded-full mr-3"></div>
                  企业级安全可靠
                </div>
                <div className="flex items-center text-white/80">
                  <div className="w-2 h-2 bg-white/60 rounded-full mr-3"></div>
                  数据驱动决策
                </div>
              </div>
            </div>
          </motion.div>
        </div>
      </div>
      
      {/* 右侧登录表单区域 */}
      <div className="flex-1 flex items-center justify-center px-4 sm:px-6 lg:px-8 bg-gray-50">
        <motion.div
          initial={{ opacity: 0, x: 30 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ duration: 0.8, delay: 0.2 }}
          className="max-w-md w-full"
        >
          <div className="bg-white rounded-3xl shadow-strong px-8 py-10">
            {/* 移动端品牌标识 */}
            <div className="lg:hidden text-center mb-8">
              <div className="inline-flex items-center justify-center w-12 h-12 bg-gradient-to-br from-primary-500 to-primary-600 rounded-xl mb-4">
                <Sparkles className="w-6 h-6 text-white" />
              </div>
              <h2 className="text-2xl font-bold text-gray-900">几何原本</h2>
              <p className="text-sm text-gray-500 mt-1">云端开发协作平台</p>
            </div>
            
            <div>
              <h2 className="text-3xl font-bold text-gray-900 mb-2">欢迎回来</h2>
              <p className="text-gray-600 mb-8">请登录您的账户以继续使用</p>
            </div>
            
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* 邮箱输入框 */}
              <div>
                <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-2">
                  邮箱地址
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Mail className="h-5 w-5 text-gray-400" />
                  </div>
                  <input
                    id="email"
                    name="email"
                    type="email"
                    value={formData.email}
                    onChange={handleInputChange}
                    className={`block w-full pl-10 pr-3 py-3 border rounded-xl shadow-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-all duration-200 ${
                      errors.email ? 'border-red-300 bg-red-50' : 'border-gray-300'
                    }`}
                    placeholder="您的邮箱地址"
                  />
                </div>
                {errors.email && (
                  <motion.p
                    initial={{ opacity: 0, y: -10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="mt-1 text-sm text-red-600"
                  >
                    {errors.email}
                  </motion.p>
                )}
              </div>
              
              {/* 密码输入框 */}
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-2">
                  密码
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Lock className="h-5 w-5 text-gray-400" />
                  </div>
                  <input
                    id="password"
                    name="password"
                    type={showPassword ? 'text' : 'password'}
                    value={formData.password}
                    onChange={handleInputChange}
                    className={`block w-full pl-10 pr-10 py-3 border rounded-xl shadow-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-all duration-200 ${
                      errors.password ? 'border-red-300 bg-red-50' : 'border-gray-300'
                    }`}
                    placeholder="您的密码"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600 transition-colors"
                  >
                    {showPassword ? (
                      <EyeOff className="h-5 w-5" />
                    ) : (
                      <Eye className="h-5 w-5" />
                    )}
                  </button>
                </div>
                {errors.password && (
                  <motion.p
                    initial={{ opacity: 0, y: -10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="mt-1 text-sm text-red-600"
                  >
                    {errors.password}
                  </motion.p>
                )}
              </div>
              
              {/* 记住我和忘记密码 */}
              <div className="flex items-center justify-between">
                <div className="flex items-center">
                  <input
                    id="rememberMe"
                    name="rememberMe"
                    type="checkbox"
                    checked={formData.rememberMe}
                    onChange={handleInputChange}
                    className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                  />
                  <label htmlFor="rememberMe" className="ml-2 block text-sm text-gray-700">
                    记住我
                  </label>
                </div>
                
                <div className="text-sm">
                  <a href="#" className="text-primary-600 hover:text-primary-500 transition-colors">
                    忘记密码？
                  </a>
                </div>
              </div>
              
              {/* 登录按钮 */}
              <motion.button
                type="submit"
                disabled={loginMutation.isPending}
                whileTap={{ scale: 0.98 }}
                className="group relative w-full flex justify-center py-3 px-4 border border-transparent text-sm font-medium rounded-xl text-white bg-gradient-to-r from-primary-600 to-primary-700 hover:from-primary-700 hover:to-primary-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-200 shadow-lg hover:shadow-xl"
              >
                {loginMutation.isPending ? (
                  <div className="flex items-center">
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    登录中...
                  </div>
                ) : (
                  <div className="flex items-center">
                    登录
                    <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
                  </div>
                )}
              </motion.button>
            </form>
            
            {/* 注册链接 */}
            <div className="mt-8 text-center">
              <p className="text-sm text-gray-600">
                还没有账户？{' '}
                <a href="#" className="text-primary-600 hover:text-primary-500 font-medium transition-colors">
                  立即注册
                </a>
              </p>
            </div>
          </div>
        </motion.div>
      </div>
    </div>
  );
};

export default Login;