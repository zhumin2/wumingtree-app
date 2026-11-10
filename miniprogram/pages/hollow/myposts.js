// 我的树洞：自己发过的帖子（含匿名帖显示为自己的记录——红线）；长按删帖。
const { request } = require('../../utils/api.js');
const { moodLabel } = require('../../utils/mood.js');
const { fmtShort } = require('../../utils/format.js');

Page({
  data: {
    posts: [],
    loading: true,
    error: '',
  },

  onLoad() {
    this.fetch();
  },

  fetch() {
    this.setData({ loading: true, error: '' });
    request('/me/posts')
      .then((data) => {
        const items = (data.items || []).map((p) => {
          p.moodLabel = moodLabel(p.mood);
          p.timeLabel = fmtShort(p.publishedAt);
          return p;
        });
        this.setData({ posts: items });
      })
      .catch((e) => this.setData({ error: e.message || '加载失败' }))
      .then(() => this.setData({ loading: false }));
  },

  onPostTap(e) {
    wx.navigateTo({ url: '/pages/hollow/detail?id=' + e.currentTarget.dataset.id });
  },

  onPostLongPress(e) {
    const id = e.currentTarget.dataset.id;
    wx.showModal({
      title: '删除这条帖子？',
      content: '回应会一并隐藏，删除后不可恢复。',
      confirmText: '删除',
      confirmColor: '#E85B3E',
      success: (res) => {
        if (!res.confirm) return;
        request('/posts/' + id, { method: 'DELETE' })
          .then(() => {
            wx.showToast({ title: '已删除', icon: 'none' });
            this.fetch();
          })
          .catch((e) => wx.showToast({ title: e.message || '删除失败', icon: 'none' }));
      },
    });
  },
});
