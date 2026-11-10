// 审核状态页：三步进度（禁返回）。审核通过后凭手机号短信验证码登录。
Page({
  data: {
    steps: [
      { title: '提交邀请码与称呼', done: true },
      { title: '管理员人工审核（1-2 天）', done: false },
      { title: '进入社群', done: false },
    ],
  },

  goLogin() {
    wx.redirectTo({ url: '/pages/login/login' });
  },
});
