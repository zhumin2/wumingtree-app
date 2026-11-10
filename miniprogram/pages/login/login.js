// 短信验证码登录：outcome 四路分流（member/home、pending/pending、
// rejected/pending、no_record/gate 提示）。审核通过即激活账号。
const { request } = require('../../utils/api.js');

Page({
  data: {
    phone: '',
    code: '',
    countdown: 0,
    sending: false,
    logging: false,
  },

  onPhone(e) { this.setData({ phone: e.detail.value }); },
  onCode(e) { this.setData({ code: e.detail.value }); },

  sendCode() {
    const phone = this.data.phone.trim();
    if (!/^1[3-9]\d{9}$/.test(phone)) {
      wx.showToast({ title: '请填写正确的手机号', icon: 'none' });
      return;
    }
    if (this.data.sending || this.data.countdown > 0) return;
    this.setData({ sending: true });
    request('/auth/sms-codes', { method: 'POST', body: { phone: phone } })
      .then(() => {
        this.setData({ countdown: 60 });
        const timer = setInterval(() => {
          if (this.data.countdown <= 1) {
            clearInterval(timer);
          }
          this.setData({ countdown: Math.max(0, this.data.countdown - 1) });
          if (this.data.countdown <= 0) clearInterval(timer);
        }, 1000);
        wx.showToast({ title: '验证码已发送', icon: 'none' });
      })
      .catch((e) => wx.showToast({ title: e.message || '发送失败', icon: 'none' }))
      .then(() => this.setData({ sending: false }));
  },

  login() {
    const phone = this.data.phone.trim();
    const code = this.data.code.trim();
    if (!/^1[3-9]\d{9}$/.test(phone)) { wx.showToast({ title: '请填写正确的手机号', icon: 'none' }); return; }
    if (code.length !== 6) { wx.showToast({ title: '请填写 6 位验证码', icon: 'none' }); return; }
    if (this.data.logging) return;
    this.setData({ logging: true });
    request('/auth/login', { method: 'POST', body: { phone: phone, code: code } })
      .then((res) => {
        if (res.outcome === 'member') {
          this.setTokens(res.tokens);
          wx.reLaunch({ url: '/pages/home/home' });
        } else if (res.outcome === 'application_pending' || res.outcome === 'application_rejected') {
          wx.redirectTo({ url: '/pages/pending/pending' });
        } else {
          wx.showToast({ title: '这个手机号还没有申请记录，请先用邀请码进入', icon: 'none' });
        }
      })
      .catch((e) => wx.showToast({ title: e.message || '登录失败', icon: 'none' }))
      .then(() => this.setData({ logging: false }));
  },

  setTokens(tokens) {
    const { setTokens } = require('../../utils/api.js');
    setTokens(tokens.accessToken, tokens.refreshToken);
  },

  goGate() {
    wx.redirectTo({ url: '/pages/gate/gate' });
  },
});
