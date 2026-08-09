const WECHAT_CODE_TIMEOUT_MS = 5000;

export class WechatCodeError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "WechatCodeError";
  }
}

/** 获取一次性微信 code。调用方只能立即发送给 Hospital 后端，不能保存或输出。 */
export function getWechatLoginCode(): Promise<string> {
  return new Promise((resolve, reject) => {
    let finished = false;

    const finish = (callback: () => void) => {
      if (finished) {
        return;
      }

      finished = true;
      clearTimeout(timer);
      callback();
    };

    const timer = setTimeout(() => {
      finish(() => reject(new WechatCodeError("获取微信登录凭证超时")));
    }, WECHAT_CODE_TIMEOUT_MS);

    uni.login({
      provider: "weixin",
      success: (result) => {
        finish(() => {
          if (result.code?.trim()) {
            resolve(result.code);
            return;
          }

          reject(new WechatCodeError("微信未返回有效登录凭证"));
        });
      },
      fail: () => {
        finish(() => reject(new WechatCodeError("获取微信登录凭证失败")));
      },
    });
  });
}
