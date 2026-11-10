// 时间与文本格式化（与客户端 friendlyTime 口径对齐）。
function pad(n) {
  return n < 10 ? '0' + n : '' + n;
}

// MM-DD HH:mm
function fmtShort(iso) {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  return pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + ' ' + pad(d.getHours()) + ':' + pad(d.getMinutes());
}

// MM-DD
function fmtDate(iso) {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  return pad(d.getMonth() + 1) + '-' + pad(d.getDate());
}

// 已加入天数（含当天）
function joinedDays(iso) {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return 1;
  return Math.floor((Date.now() - d.getTime()) / 86400000) + 1;
}

// 手机号掩码（服务端已返回掩码串；此处兜底）
function maskPhone(p) {
  if (!p || p.length !== 11) return '';
  return p.slice(0, 3) + '****' + p.slice(7);
}

module.exports = {
  fmtShort: fmtShort,
  fmtDate: fmtDate,
  joinedDays: joinedDays,
  maskPhone: maskPhone,
};
