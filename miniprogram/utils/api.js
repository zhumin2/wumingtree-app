// 统一信封请求层（架构设计 3.2/4.4 的小程序版）：
// - 2001 → 静默续期后重放一次
// - 2002 → 清会话并回登录页
// - 其余非零 code 抛 { code, message } 供页面展示
const { BASE_URL } = require('./config.js');

const TOKEN_KEY = 'wmt.access';
const REFRESH_KEY = 'wmt.refresh';

function getToken() {
  try {
    return wx.getStorageSync(TOKEN_KEY) || '';
  } catch (e) {
    return '';
  }
}

function setTokens(access, refresh) {
  if (access) wx.setStorageSync(TOKEN_KEY, access);
  if (refresh) wx.setStorageSync(REFRESH_KEY, refresh);
}

function clearTokens() {
  try {
    wx.removeStorageSync(TOKEN_KEY);
    wx.removeStorageSync(REFRESH_KEY);
  } catch (e) {
    // 忽略存储异常
  }
}

let refreshing = false;

function rawRequest(path, opts) {
  const options = opts || {};
  return new Promise((resolve, reject) => {
    const header = { 'Content-Type': 'application/json', Accept: 'application/json' };
    const token = getToken();
    if (token) header.Authorization = 'Bearer ' + token;
    if (options.headers) {
      for (const k in options.headers) header[k] = options.headers[k];
    }
    wx.request({
      url: BASE_URL + path,
      method: options.method || 'GET',
      data: options.body,
      header: header,
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data);
        } else {
          reject(new Error('网络异常 ' + res.statusCode));
        }
      },
      fail: () => reject(new Error('网络不太顺畅，稍后再试试')),
    });
  });
}

function refresh() {
  if (refreshing) return Promise.resolve(false);
  const rt = wx.getStorageSync(REFRESH_KEY);
  if (!rt) return Promise.resolve(false);
  refreshing = true;
  return rawRequest('/auth/refresh', { method: 'POST', body: { refreshToken: rt } })
    .then((env) => {
      if (env.code === 0 && env.data && env.data.accessToken) {
        setTokens(env.data.accessToken, env.data.refreshToken);
        return true;
      }
      return false;
    })
    .catch(() => false)
    .then((ok) => {
      refreshing = false;
      return ok;
    });
}

function forceLogout() {
  clearTokens();
  wx.reLaunch({ url: '/pages/login/login' });
}

// request 返回信封 data；非零 code 抛业务错误对象。
function request(path, opts, retried) {
  return rawRequest(path, opts).then((env) => {
    if (env.code === 0) return env.data;
    if (env.code === 2001 && !retried) {
      return refresh().then((ok) => {
        if (!ok) {
          forceLogout();
          throw { code: 2001, message: '登录已过期，请重新登录' };
        }
        return request(path, opts, true);
      });
    }
    if (env.code === 2002) {
      forceLogout();
      throw { code: 2002, message: env.message || '登录已过期，请重新登录' };
    }
    throw { code: env.code, message: env.message || '错误码 ' + env.code };
  });
}

module.exports = {
  request: request,
  rawRequest: rawRequest,
  getToken: getToken,
  setTokens: setTokens,
  clearTokens: clearTokens,
};
