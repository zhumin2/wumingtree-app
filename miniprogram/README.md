# miniprogram · 无名树小程序端（微信原生）

邀请制家长陪伴社群的微信小程序客户端。与 `app/`（Flutter）共用同一 Go API 与统一信封约定，登录沿用「手机号 + 短信验证码」；wx.login/code2Session 身份打通需要 AppID 与企业资质，列第二阶段。

## 目录

- `app.json / app.js / app.wxss`：全局配置、会话守卫入口、设计 token（色板与技术设计 3.3 一致）
- `utils/`：`api.js`（信封解包 + 2001 静默续期 + 2002 吊销回登录）、`mood.js`、`format.js`
- `pages/`：gate（邀请进入）、pending（审核状态）、login、home（四入口 + 常驻风险提示）、hollow（列表/发帖/详情/我的树洞）、me（身份卡 + 设置）
- `tools/check.js`：结构校验（pages 与文件对应、JSON 可解析、红线静态检查）

## 导入与运行

1. 微信开发者工具 → 导入 `miniprogram/` 目录（AppID 用 `touristappid` 测试号或替换为已注册的小程序 AppID）。
2. 联调：`utils/config.js` 改 BASE_URL 为 `http://127.0.0.1:8080/api/v1`，详情设置勾选「不校验合法域名」。
3. mock 短信：服务端 `SMS_TYPE=mock` 时验证码固定 123456。

## 上线前必办

- `utils/config.js` 的 BASE_URL 改为正式 HTTPS 域名（需 ICP 备案），并在小程序后台配置 request 合法域名。
- 用户隐私保护指引、`wx.getStorageSync` 隐私接口声明在小程序后台提交。
- UGC 合规：服务端机审 + 举报入口已就绪（小红书式平台审核以小程序类目要求为准）。

## 红线落实（与 app/ 一致）

- 树洞默认匿名；红点无数字；时间倒序无热度排序
- 危机词不拦截，详情页叠加求助卡；首页常驻风险提示
- 举报入口 + sitemap 全量 disallow（不做公开分发）
- 手机号仅服务端可见（掩码串回显）
