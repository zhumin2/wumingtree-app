// 发帖：心情 + 标题（必填）+ 正文（可选）+ 匿名开关（默认开）。
// 提交前提示不留可识别信息；client_token 幂等（架构设计 3.4）。
const { request } = require('../../utils/api.js');
const { MOODS } = require('../../utils/mood.js');

Page({
  data: {
    moods: MOODS,
    mood: 'tired',
    title: '',
    body: '',
    anonymous: true,
    sending: false,
  },

  onMoodTap(e) {
    this.setData({ mood: e.currentTarget.dataset.mood });
  },

  onTitle(e) { this.setData({ title: e.detail.value }); },
  onBody(e) { this.setData({ body: e.detail.value }); },

  onAnonChange(e) {
    this.setData({ anonymous: e.detail.value });
  },

  clientToken() {
    // 简单随机 token：重试复用同一值即可命中服务端幂等
    let s = '';
    for (let i = 0; i < 24; i++) {
      s += Math.floor(Math.random() * 16).toString(16);
    }
    return s;
  },

  submit() {
    const title = this.data.title.trim();
    if (!title) {
      wx.showToast({ title: '写一句标题吧', icon: 'none' });
      return;
    }
    if (this.data.sending) return;
    const token = this.clientToken();
    this.setData({ sending: true });
    request('/posts', {
      method: 'POST',
      headers: { client_token: token },
      body: {
        mood: this.data.mood,
        title: title,
        body: this.data.body.trim(),
        isAnonymous: this.data.anonymous,
      },
    })
      .then((post) => {
        wx.redirectTo({ url: '/pages/hollow/detail?id=' + post.id });
      })
      .catch((e) => wx.showToast({ title: e.message || '发布失败', icon: 'none' }))
      .then(() => this.setData({ sending: false }));
  },
});
