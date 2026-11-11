// 协议静态页：?type=user|privacy|rules（默认 user，深链兜底）。
// 正文常量见 content.js；样式只用 app.wxss 的 --wmt-* 变量。
const { getDoc } = require('./content.js');

Page({
  data: {
    title: '',
    sections: [],
  },

  onLoad(query) {
    const doc = getDoc(query && query.type);
    this.setData({ title: doc.title, sections: doc.sections });
    wx.setNavigationBarTitle({ title: doc.title });
  },
});
