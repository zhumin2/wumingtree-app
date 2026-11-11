// 邀请进入页：邀请码 + 称呼 + 手机号 → POST /applications → 审核状态页。
// 提交后由管理员人工审核（1-2 天）；不会推送营销信息。
const { request } = require('../../utils/api.js');

Page({
  data: {
    invite: '',
    nickname: '',
    phone: '',
    submitting: false,
    agreed: false,
  },

  onInvite(e) { this.setData({ invite: e.detail.value }); },
  onNickname(e) { this.setData({ nickname: e.detail.value }); },
  onPhone(e) { this.setData({ phone: e.detail.value }); },

  toggleAgree() {
    this.setData({ agreed: !this.data.agreed });
  },

  openAgreement(e) {
    wx.navigateTo({ url: '/pages/agreement/agreement?type=' + (e.currentTarget.dataset.type || 'user') });
  },

  submit() {
    const invite = this.data.invite.trim();
    const nickname = this.data.nickname.trim();
    const phone = this.data.phone.trim();
    if (!invite) { wx.showToast({ title: '请输入邀请码', icon: 'none' }); return; }
    if (!nickname) { wx.showToast({ title: '请填写称呼（最多 8 字）', icon: 'none' }); return; }
    if (!/^1[3-9]\d{9}$/.test(phone)) { wx.showToast({ title: '请填写正确的手机号', icon: 'none' }); return; }
    if (!this.data.agreed) { wx.showToast({ title: '请先阅读并勾选同意协议', icon: 'none' }); return; }
    this.setData({ submitting: true });
    request('/applications', { method: 'POST', body: { inviteCode: invite, nickname: nickname, phone: phone } })
      .then(() => {
        wx.redirectTo({ url: '/pages/pending/pending' });
      })
      .catch((e) => {
        wx.showToast({ title: e.message || '提交失败', icon: 'none' });
      })
      .then(() => this.setData({ submitting: false }));
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' });
  },
});
