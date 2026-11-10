// 帖子与楼层：crisis 求助卡（红线）、抱一下/我也是 toggle、楼层回应、举报抽屉。
const { request } = require('../../utils/api.js');
const { moodLabel } = require('../../utils/mood.js');
const { fmtShort } = require('../../utils/format.js');

const REPORT_REASONS = ['泄露了孩子或家庭的隐私', '带有评判、指责或攻击', '广告、引流或与社群无关', '其他'];

Page({
  data: {
    post: null,
    floors: [],
    crisis: false,
    loading: true,
    error: '',
    floorInput: '',
    anonFloor: true,
    sending: false,
    reasonIndex: -1,
  },

  onLoad(options) {
    this.postId = parseInt(options.id, 10);
    this.fetch();
  },

  fetch() {
    this.setData({ loading: true, error: '' });
    return request('/posts/' + this.postId)
      .then((d) => {
        const post = d.post;
        post.moodLabel = moodLabel(post.mood);
        post.timeLabel = fmtShort(post.publishedAt);
        const floors = (d.floors || []).map((f) => {
          f.timeLabel = fmtShort(f.createdAt);
          return f;
        });
        this.setData({ post: post, floors: floors, crisis: d.crisis === true });
      })
      .catch((e) => this.setData({ error: e.message || '加载失败' }))
      .then(() => this.setData({ loading: false }));
  },

  togglePostReaction(e) {
    const type = e.currentTarget.dataset.type; // hug | me_too
    request('/posts/' + this.postId + '/reactions', { method: 'POST', body: { type: type } })
      .then((st) => {
        const p = this.data.post;
        const my = (p.myReactions || []).filter((t) => t !== type);
        if (st.active) my.push(type);
        p.myReactions = my;
        if (type === 'hug') p.hug = st.count;
        if (type === 'me_too') p.meToo = st.count;
        this.setData({ post: p });
      })
      .catch((e) => wx.showToast({ title: e.message || '操作失败', icon: 'none' }));
  },

  toggleFloorHug(e) {
    const floorId = e.currentTarget.dataset.id;
    request('/floors/' + floorId + '/reactions', { method: 'POST', body: { type: 'hug' } })
      .then(() => this.fetch())
      .catch((e) => wx.showToast({ title: e.message || '操作失败', icon: 'none' }));
  },

  onFloorInput(e) { this.setData({ floorInput: e.detail.value }); },
  toggleAnon() { this.setData({ anonFloor: !this.data.anonFloor }); },

  sendFloor() {
    const content = this.data.floorInput.trim();
    if (!content || this.data.sending) return;
    this.setData({ sending: true });
    request('/posts/' + this.postId + '/floors', {
      method: 'POST',
      headers: { client_token: 'floor-' + Date.now() + '-' + Math.floor(Math.random() * 1e8).toString(16) },
      body: { content: content, isAnonymous: this.data.anonFloor },
    })
      .then(() => {
        this.setData({ floorInput: '' });
        this.fetch();
      })
      .catch((e) => wx.showToast({ title: e.message || '发送失败', icon: 'none' }))
      .then(() => this.setData({ sending: false }));
  },

  report() {
    wx.showActionSheet({
      itemList: REPORT_REASONS,
      success: (res) => {
        const reason = ['privacy', 'attack', 'ads', 'other'][res.tapIndex];
        request('/reports', {
          method: 'POST',
          body: { targetType: 'post', targetId: this.postId, reason: reason },
        })
          .then(() => wx.showToast({ title: '已提交举报，管理员会处理', icon: 'none' }))
          .catch((e) => wx.showToast({ title: e.message || '举报失败', icon: 'none' }));
      },
      fail: () => {},
    });
  },
});
