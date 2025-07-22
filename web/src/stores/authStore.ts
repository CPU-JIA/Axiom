import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface User {
  id: string;
  email: string;
  name: string;
  avatar?: string;
  role: 'admin' | 'user' | 'viewer';
  tenantId: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  token: string | null;
}

interface AuthActions {
  login: (email: string, password: string) => Promise<boolean>;
  logout: () => void;
  setUser: (user: User) => void;
  setToken: (token: string) => void;
  initializeAuth: () => void;
}

type AuthStore = AuthState & AuthActions;

export const useAuthStore = create<AuthStore>()(
  persist(
    (set, get) => ({
      // State
      user: null,
      isAuthenticated: false,
      isLoading: true,
      token: null,

      // Actions
      login: async (email: string, password: string): Promise<boolean> => {
        try {
          set({ isLoading: true });
          
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 1500));
          
          // 模拟登录成功
          if (email === 'jia@euclid.com' && password === 'password123') {
            const mockUser: User = {
              id: '1',
              email: 'jia@euclid.com',
              name: 'JIA',
              role: 'admin',
              tenantId: 'tenant-1',
              avatar: 'https://ui-avatars.com/api/?name=JIA&background=0ea5e9&color=fff'
            };
            
            const mockToken = 'mock-jwt-token-' + Date.now();
            
            set({
              user: mockUser,
              token: mockToken,
              isAuthenticated: true,
              isLoading: false,
            });
            
            return true;
          } else {
            set({ isLoading: false });
            return false;
          }
        } catch (error) {
          console.error('Login error:', error);
          set({ isLoading: false });
          return false;
        }
      },

      logout: () => {
        set({
          user: null,
          token: null,
          isAuthenticated: false,
          isLoading: false,
        });
        
        // 清除其他相关状态
        localStorage.removeItem('auth-storage');
      },

      setUser: (user: User) => {
        set({ user, isAuthenticated: true });
      },

      setToken: (token: string) => {
        set({ token });
      },

      initializeAuth: () => {
        const state = get();
        
        // 检查是否有有效的token和用户信息
        if (state.token && state.user) {
          set({
            isAuthenticated: true,
            isLoading: false,
          });
        } else {
          set({
            isAuthenticated: false,
            isLoading: false,
          });
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);