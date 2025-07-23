// Axios类型扩展
declare module 'axios' {
  export interface AxiosRequestConfig {
    metadata?: {
      startTime: Date;
    };
    params?: Record<string, unknown>;
  }
}