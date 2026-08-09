/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly DEV: boolean;
  readonly VITE_API_BASE_URL?: string;
  readonly VITE_API_TRANSPORT?: "direct" | "cloudbase";
  readonly VITE_CLOUDBASE_ENV_ID?: string;
  readonly VITE_ANYSERVICE_NAME?: string;
  readonly VITE_STAFF_DATA_SOURCE?: "mock" | "http";
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare module "*.vue" {
  import type { DefineComponent } from "vue";

  const component: DefineComponent<Record<string, never>, Record<string, never>, unknown>;
  export default component;
}
