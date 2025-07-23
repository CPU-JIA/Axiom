import React, { useEffect } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { AnimatePresence } from 'framer-motion';
import Login from './pages/Login';
import EnhancedLogin from './pages/EnhancedLogin';
import Dashboard from './pages/Dashboard';
import ProjectManagement from './pages/ProjectManagement';
import TaskBoard from './pages/TaskBoard';
import Settings from './pages/Settings';
import TestLogin from './components/TestLogin';
import ErrorBoundary from './components/ErrorBoundary';
import { useAuthStore } from './stores/authStore';
import { NotificationProvider } from './components/NotificationSystem';

// 私有路由组件
interface PrivateRouteProps {
  children: React.ReactNode;
}

const PrivateRoute: React.FC<PrivateRouteProps> = ({ children }) => {
  const { isAuthenticated, isLoading } = useAuthStore();
  
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    );
  }
  
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />;
};

// 公开路由组件（已认证用户会被重定向到dashboard）
interface PublicRouteProps {
  children: React.ReactNode;
}

const PublicRoute: React.FC<PublicRouteProps> = ({ children }) => {
  const { isAuthenticated, isLoading } = useAuthStore();
  
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    );
  }
  
  return !isAuthenticated ? <>{children}</> : <Navigate to="/dashboard" replace />;
};

const App: React.FC = () => {
  const { initializeAuth } = useAuthStore();
  
  useEffect(() => {
    // 应用启动时初始化认证状态
    initializeAuth();
  }, [initializeAuth]);

  return (
    <ErrorBoundary>
      <NotificationProvider>
        <div className="App">
          <AnimatePresence mode="wait">
            <Routes>
              {/* 公开路由 */}
              <Route 
                path="/login" 
                element={
                  <PublicRoute>
                    <EnhancedLogin />
                  </PublicRoute>
                } 
              />
              
              {/* 原始登录页面（备用） */}
              <Route 
                path="/login-original" 
                element={
                  <PublicRoute>
                    <Login />
                  </PublicRoute>
                } 
              />
              
              {/* 测试登录页面 */}
              <Route 
                path="/test-login" 
                element={<TestLogin />} 
              />
              
              {/* 私有路由 */}
              <Route 
                path="/dashboard" 
                element={
                  <PrivateRoute>
                    <Dashboard />
                  </PrivateRoute>
                } 
              />
              
              <Route 
                path="/projects" 
                element={
                  <PrivateRoute>
                    <ProjectManagement />
                  </PrivateRoute>
                } 
              />
              
              <Route 
                path="/tasks" 
                element={
                  <PrivateRoute>
                    <TaskBoard />
                  </PrivateRoute>
                } 
              />
              
              <Route 
                path="/settings" 
                element={
                  <PrivateRoute>
                    <Settings />
                  </PrivateRoute>
                } 
              />
              
              {/* 默认重定向 */}
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              
              {/* 404 页面 */}
              <Route 
                path="*" 
                element={
                  <div className="min-h-screen flex items-center justify-center bg-gray-50">
                    <div className="text-center">
                      <h1 className="text-4xl font-bold text-gray-900 mb-4">404</h1>
                      <p className="text-gray-600 mb-8">页面未找到</p>
                      <a 
                        href="/dashboard" 
                        className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-lg shadow-sm text-white bg-primary-600 hover:bg-primary-700"
                      >
                        返回首页
                      </a>
                    </div>
                  </div>
                } 
              />
            </Routes>
          </AnimatePresence>
        </div>
      </NotificationProvider>
    </ErrorBoundary>
  );
};

export default App;