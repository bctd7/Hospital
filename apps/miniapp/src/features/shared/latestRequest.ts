// 页面无法真正取消所有小程序请求，但可以保证过期响应不再覆盖用户的新选择。
export interface LatestRequestGuard {
  begin: () => number;
  cancel: () => void;
  isCurrent: (token: number) => boolean;
}

export function createLatestRequestGuard(): LatestRequestGuard {
  let sequence = 0;
  return {
    begin() {
      sequence += 1;
      return sequence;
    },
    cancel() {
      sequence += 1;
    },
    isCurrent(token: number) {
      return token === sequence;
    },
  };
}
