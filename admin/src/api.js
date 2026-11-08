// 统一信封解包（架构设计 3.2）；2001/2002 清会话回登录。
const TOKEN_KEY = 'wmt.admin.token'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(t) {
  if (t) localStorage.setItem(TOKEN_KEY, t)
  else localStorage.removeItem(TOKEN_KEY)
}

export async function api(path, { method = 'GET', body } = {}) {
  const headers = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers.Authorization = 'Bearer ' + token
  const res = await fetch('/api/v1' + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const env = await res.json()
  if (env.code === 0) return env.data
  if (env.code === 2001 || env.code === 2002) {
    setToken(null)
    if (!location.pathname.endsWith('/login')) location.reload()
  }
  throw new Error(env.message || '错误码 ' + env.code)
}
