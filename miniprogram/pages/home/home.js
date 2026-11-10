// 首页：会话守卫 + 四入口（仅树洞已开放，其余「即将开放」）+ 常驻风险提示（红线）。
const { getToken } = require('../../utils/api.js');

Page({
  data: {
    greeting: '',
  },

  onShow() {
    if (!getToken()) {
      wx.reLaunch({ url: '/pages/gate/gate' });
      return;
    }
    const now = new Date();
    const hours = now.getHours();
    const greet = hours < 6 ? '夜深了' : hours < 12 ? '早上好' : hours < 18 ? '下午好' : '晚上好';
    this.setData({
      greeting: greet + '，今天也可以慢慢来',
    });
  },

  onHollow() {
    wx.navigateTo({ url: '/pages/hollow/list' });
  },

  onMe() {
    wx.navigateTo({ url: '/pages/me/me' });
  },

  onComingSoon() {
    wx.showToast({ title: '这个模块在接下来的版本开放', icon: 'none' });
  },
});
