const WECHAT_CODE_TIMEOUT_MS = 1800;

/**
 * 尝试获取微信临时 code，但在后端换码接口完成前不保存、不输出该 code。
 * 获取失败或超时都会正常结束，入口页面不得因此阻塞。
 */
export function attemptWechatCode(): Promise<void> {
  return new Promise((resolve) => {
    let finished = false;

    const finish = () => {
      if (finished) {
        return;
      }

      finished = true;
      clearTimeout(timer);
      resolve();
    };

    const timer = setTimeout(finish, WECHAT_CODE_TIMEOUT_MS);

    uni.login({
      provider: "weixin",
      success: finish,
      fail: finish,
    });
  });
}
