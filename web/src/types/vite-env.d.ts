/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string;
  readonly VITE_DEMO_EMAIL: string;
  readonly VITE_DEMO_PASSWORD: string;
  readonly VITE_APP_TITLE: string;
  readonly VITE_SENTRY_DSN?: string;
  readonly VITE_ENVIRONMENT: 'development' | 'staging' | 'production';
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}