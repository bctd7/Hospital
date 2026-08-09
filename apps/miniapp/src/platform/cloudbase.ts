import {
  ANYSERVICE_NAME,
  API_TRANSPORT,
  CLOUDBASE_ENV_ID,
} from "@/config/environment";

export interface CloudContainerResult {
  statusCode: number;
  data: unknown;
}

interface CloudContainerOptions {
  path: string;
  header: Record<string, string>;
  method: "GET" | "POST" | "PUT" | "DELETE";
  data?: unknown;
}

interface WeChatCloudApi {
  init(options: { env: string; traceUser: boolean }): void;
  callContainer(options: CloudContainerOptions): Promise<CloudContainerResult>;
}

interface WeChatRuntime {
  wx?: {
    cloud?: WeChatCloudApi;
  };
}

let initialized = false;

function cloudApi(): WeChatCloudApi {
  const api = (globalThis as WeChatRuntime).wx?.cloud;
  if (!api) {
    throw new Error("当前运行环境不支持微信云开发");
  }
  return api;
}

export function initializeCloudBase(): WeChatCloudApi | undefined {
  if (API_TRANSPORT !== "cloudbase") {
    return undefined;
  }
  if (!CLOUDBASE_ENV_ID || !ANYSERVICE_NAME) {
    throw new Error("CloudBase 环境 ID 或 AnyService 服务标识未配置");
  }

  const api = cloudApi();
  if (!initialized) {
    api.init({ env: CLOUDBASE_ENV_ID, traceUser: true });
    initialized = true;
  }
  return api;
}

export async function callAnyService(
  options: CloudContainerOptions,
): Promise<CloudContainerResult> {
  const api = initializeCloudBase();
  if (!api) {
    throw new Error("AnyService 只能在 CloudBase 请求模式下调用");
  }

  return api.callContainer({
    ...options,
    header: {
      ...options.header,
      "X-WX-SERVICE": "tcbanyservice",
      "X-AnyService-Name": ANYSERVICE_NAME,
    },
  });
}
