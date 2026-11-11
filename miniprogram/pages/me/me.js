// 我的：身份卡 + 我的树洞入口 + 设置（昵称/默认匿名/免打扰/提醒）+ 退出。
// 昵称 8 字上限；开关即时 PATCH /me（服务端校验，失败重渲染回原值）。
const { request, clearTokens } = require('../../utils/api.js');
const { joinedDays } = require('../../utils/format.js');

Page({
  data: {
    me: null,
    days: 0,
    loading: true,
    version: '1.0.0',
  },

  onShow() {
    this.fetchMe();
  },

  fetchMe() {
    request('/me')
      .then((me) => {
        me.days = joinedDays(me.joinedAt);
        this.setData({ me: me, loading: false });
      })
      .catch((e) => {
        // 2001/2002 已由请求层处理；此处兜底展示
        this.setData({ loading: false, error: e.message || '' });
      });
  },

  onMyPosts() {
    wx.navigateTo({ url: '/pages/hollow/myposts' });
  },

  openAgreement(e) {
    wx.navigateTo({ url: '/pages/agreement/agreement?type=' + (e.currentTarget.dataset.type || 'user') });
  },

  editNickname() {
    const me = this.data.me;
    if (!me) return;
    wx.showModal({
      title: '修改昵称',
      editable: true,
      placeholderText: '其他人看到的名字（最多 8 字）',
      content: me.nickname,
      success: (res) => {
        if (!res.confirm) return;
        const nickname = (res.content || '').trim();
        if (!nickname) return;
        this.patchMe({ nickname: nickname });
      },
    });
  },

  onDefaultAnon(e) { this.patchMe({ defaultAnon: e.detail.value }); },
  onDnd(e) { this.patchMe({ dnd: e.detail.value }); },
  onNotifyNotice(e) { this.patchMe({ notifyNotice: e.detail.value }); },
  onNotifyHollow(e) { this.patchMe({ notifyHollow: e.detail.value }); },

  patchMe(patch) {
    request('/me', { method: 'PATCH', body: patch })
      .then((me) => {
        me.days = joinedDays(me.joinedAt);
        this.setData({ me: me });
        wx.showToast({ title: '已更新', icon: 'none' });
      })
      .catch((e) => {
        wx.showToast({ title: e.message || '更新失败', icon: 'none' });
        this.fetchMe(); // 服务端为准：失败回原值
      });
  },

  logout() {
    wx.showModal({
      title: '退出登录？',
      content: '下次进入需要重新用短信验证码登录。',
      confirmText: '退出',
      confirmColor: '#E85B3E',
      success: (res) => {
        if (!res.confirm) return;
        clearTokens();
        wx.reLaunch({ url: '/pages/gate/gate' });
      },
    });
  },
});
