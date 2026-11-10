// 树洞列表：心情筛选 + 时间倒序（红线：无热度排序）+ 下拉刷新/触底加载。
const { request } = require('../../utils/api.js');
const { MOODS, moodLabel } = require('../../utils/mood.js');
const { fmtShort } = require('../../utils/format.js');

Page({
  data: {
    moods: [{ code: '', label: '全部' }].concat(MOODS),
    activeMood: '',
    posts: [],
    loading: false,
    error: '',
    cursor: '',
    hasMore: false,
  },

  onLoad() {
    this.fetch(true);
  },

  onPullDownRefresh() {
    this.fetch(true).then(() => wx.stopPullDownRefresh());
  },

  onReachBottom() {
    if (this.data.hasMore && !this.data.loading) this.fetch(false);
  },

  onMoodTap(e) {
    this.fetch(true, e.currentTarget.dataset.mood);
  },

  fetch(reset, moodArg) {
    const mood = moodArg !== undefined ? moodArg : this.data.activeMood;
    const cursor = reset ? '' : this.data.cursor;
    this.setData({ loading: true, error: '', activeMood: mood });
    const query = [];
    if (mood) query.push('mood=' + mood);
    if (cursor) query.push('cursor=' + encodeURIComponent(cursor));
    const qs = query.length ? '?' + query.join('&') : '';
    return request('/posts' + qs)
      .then((data) => {
        const items = (data.items || []).map((p) => {
          p.moodLabel = moodLabel(p.mood);
          p.timeLabel = fmtShort(p.publishedAt);
          return p;
        });
        const posts = reset ? items : this.data.posts.concat(items);
        this.setData({
          posts: posts,
          cursor: data.nextCursor || '',
          hasMore: !!data.nextCursor,
        });
      })
      .catch((e) => this.setData({ error: e.message || '加载失败' }))
      .then(() => this.setData({ loading: false }));
  },

  onPostTap(e) {
    wx.navigateTo({ url: '/pages/hollow/detail?id=' + e.currentTarget.dataset.id });
  },

  onCompose() {
    wx.navigateTo({ url: '/pages/hollow/compose' });
  },
});
