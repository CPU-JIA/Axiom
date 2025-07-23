import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../stores/authStore';

const TestLogin: React.FC = () => {
  const navigate = useNavigate();
  const [isLoading, setIsLoading] = useState(false);
  const [message, setMessage] = useState('');

  const handleTestLogin = async () => {
    setIsLoading(true);
    setMessage('开始登录...');
    
    try {
      const { login } = useAuthStore.getState();
      const success = await login('jia@euclid.com', 'password123');
      
      if (success) {
        setMessage('登录成功！正在跳转...');
        setTimeout(() => {
          navigate('/dashboard');
        }, 1000);
      } else {
        setMessage('登录失败：邮箱或密码错误');
      }
    } catch (error) {
      console.error('Login error:', error);
      setMessage('登录错误：' + String(error));
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogout = () => {
    const { logout } = useAuthStore.getState();
    logout();
    setMessage('已退出登录');
  };

  const checkAuthState = () => {
    const state = useAuthStore.getState();
    setMessage(`认证状态: ${JSON.stringify({
      isAuthenticated: state.isAuthenticated,
      isLoading: state.isLoading,
      user: state.user?.name || 'null'
    }, null, 2)}`);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 p-8">
      <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8">
        <h1 className="text-2xl font-bold text-center mb-8">登录测试页面</h1>
        
        <div className="space-y-4">
          <button
            onClick={handleTestLogin}
            disabled={isLoading}
            className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? '登录中...' : '测试登录'}
          </button>
          
          <button
            onClick={handleLogout}
            className="w-full bg-gray-600 text-white py-3 px-4 rounded-lg hover:bg-gray-700"
          >
            退出登录
          </button>
          
          <button
            onClick={checkAuthState}
            className="w-full bg-green-600 text-white py-3 px-4 rounded-lg hover:bg-green-700"
          >
            检查认证状态
          </button>
        </div>
        
        {message && (
          <div className="mt-6 p-4 bg-gray-100 rounded-lg">
            <pre className="text-sm text-gray-800 whitespace-pre-wrap">{message}</pre>
          </div>
        )}
        
        <div className="mt-6 text-sm text-gray-600">
          <p><strong>测试账户：</strong></p>
          <p>邮箱: jia@euclid.com</p>
          <p>密码: password123</p>
        </div>
      </div>
    </div>
  );
};

export default TestLogin;