# 无名树 App · Agent 接手说明

> 最后更新：2026-11-08
> 仓库：https://github.com/zhumin2/wumingtree-app
> 在线预览：https://zhumin2.github.io/wumingtree-app/
> 本文档写给接手的 agent、开发者和设计者。请先读完本文，再打开原型或改代码。

## 0. 一分钟了解

无名树是一个**只对邀请开放的家长陪伴社群**，服务处在休整期（休学、居家、拒学）的孩子与家庭。它不是泛社交产品，也不是教育机构招生页，而是一个让家长先安全说出处境、再找到可复用经验和真实活动的地方。

三条不能丢的价值观：

- 不排名
- 不打卡
- 不比孩子

产品当前的核心资产是一份**单文件 HTML 高保真原型** `index.html`，以及一份 Word 版《无名树 App 产品说明文档》。原型里的数据全部是虚构示例，刷新即复位。

## 1. 当前交付物

| 文件 | 作用 | 备注 |
|---|---|---|
| `index.html` | v3 明亮活泼版可交互原型 | 单文件，无外链依赖，双击可打开 |
| `preview.png` | 原型首页截图 | 1440×900，用于 README 和文档配图 |
| `README.md` | 仓库首页说明 | 含在线预览、演示路径、产品红线、演示边界 |
| `docs/无名树App产品说明文档-v1.0.docx` | 产品说明文档 | 12 章、34 张表、1 张首页截图；正式评审稿 |
| `agents.md` | 本接手说明 | 记录现状、坑、构建与发布方式 |
| `docs/无名树App技术设计文档-v1.0.md` | 产品技术设计 | Flutter + Go + MySQL；数据模型、API、权限、安全、里程碑 |
| `docs/无名树App技术架构设计-v1.0.md` | 技术架构定稿 | 沿用「拾光 ShinePick」生产验证模式；工程约定唯一来源 |
| `docs/tech/无名树App细化设计-v1.0.md` | 细化设计 | 状态机、错误码、校验、限流、定时任务 |
| `docs/tech/schema-init-v1.0.sql` | 数据库 DDL v1.1 | 19 张表 MySQL 8；配套 golang-migrate up/down |
| `docs/tech/openapi-v1.0.yaml` | API 契约初稿 | 评审与联调参考；实现走统一信封 + 手写 VO |
| `docs/无名树App开发计划-v1.0.md` | 开发计划 | M0–M6 任务分解（111 人天）、门禁、长周期事项 |
| `server/` | Go API 服务端 | M0–M2 已建：信封与错误码、短信登录、邀请准入、我的模块、审核端点、树洞全链路（发帖/楼层/回应/举报/机审钩子/危机分支）、seedadmin；部署三件套在 `server/deploy/linux/` |
| `app/` | Flutter 客户端 | M0–M2 已建：邀请/登录/我的/设置 + 树洞列表/发帖/详情/我的树洞 + M1-10 设计系统组件；平台目录初始化见 `app/README.md` |
| `admin/` | 管理台 React + Ant Design | M1-6 最小版已建：短信登录、申请列表/筛选、通过/拒绝（原因模板）；dev 经 vite 代理本机 API，`npm run build` 已验证 |
| `app/`（Web 形态） | 用户端 Web | 同一代码库 `flutter build web`，`/static/web/` 由 API 同源托管（no-cache）；CI 产出 web-dist artifact |
| `miniprogram/` | 微信小程序端（原生） | MP1 已建：邀请/审核/登录/首页/树洞四页/我的/协议页，复用 Go API 信封约定；`node tools/check.js` 结构校验通过；导入见 `miniprogram/README.md` |
| `docs/agreements/` | 协议正本 | 用户协议 / 隐私政策 / 社区规范 v1（客户端页面与本文件同步）；替换正式法务文本只改此处与两端常量 |

线上 Pages 由 `main` 分支根目录直接发布，`index.html` 即线上首页。改动 `index.html` 后，Pages 会自动重建；通常 1—2 分钟内生效。

## 2. 产品定位与用户

### 2.1 一句话

> 无名树是一个只对邀请开放的家长陪伴社群，服务处在休整期（休学、居家、拒学）的孩子与家庭，用匿名互助、长期经验和真实活动，把“撑过这一段”变成“有人一起走”。

### 2.2 用户

- **新加入家长**：刚被邀请，仍在观望，需要先确认这里安全。
- **长期陪伴家长**：已经经历过一段休整期，愿意留下经验。
- **活动型家庭**：愿意让孩子尝试线下活动，但怕尴尬。
- **管理员**：负责邀请审核、公告、举报与秩序。
- **带队老师**：负责活动执行、讨论区与老师手记。
- **孩子**：间接受益者，不直接注册、不直接发帖、不出现在公开页面。

### 2.3 非目标

不做公开广场、算法推荐、排行榜、积分、打卡、晒娃 feed、商业广告、家长间陌生人社交、医疗诊断或治疗承诺。

## 3. 原型现状

### 3.1 规模

- 21 个视图、30 个状态用例。
- 4 个底部 Tab：首页、树洞、活动、我的。
- 4 个首页入口：公告板（黄）、树洞互助（珊瑚）、经验资源库（蓝）、活动专区（绿）。
- 5 场活动：野外生存、PBL 共建、文学戏剧、传统文化、家长沙龙。
- 4 条树洞帖、5 个资源文件夹、3 条公告、2 个私信联系人。
- 默认进入首页；左侧说明卡“01 邀请进入”可走完整申请与模拟审核流程。

### 3.2 演示路径

1. 邀请进入 → 提交申请 → 模拟管理员审核 → 首页
2. 首页 → 公告板 → 公告详情
3. 首页 → 树洞 → 心情筛选 → 发帖 → 帖子与楼层 → 回应／举报
4. 首页 → 经验资源库 → 文件夹 → 文件提示
5. 首页 → 活动专区 → 活动详情（详情／讨论区／活动档案）
6. 活动详情 → 报名 → 报名成功 → 我的报名 → 档案解锁
7. 我的 → 我的树洞／孩子的作品／私信／设置
8. 活动档案未报名状态 → 锁定页（真实产品建议直接隐藏 Tab）

### 3.3 产品红线（改代码前必须确认没有被破坏）

- 公告板只读，不开放评论。
- 树洞是“楼”不是聊天流；默认匿名；红点不显示数字；不做“已读”统计。
- 不发公开广场，不做热度排序、排行榜、积分、勋章、打卡。
- 活动档案报名后才可见；相册只显示集体合照与本人孩子；禁止保存转发。
- 老师手记不打分、不评级，只记录发生了什么。
- 首页常驻风险提示：同伴互助不能替代专业医疗或心理支持。
- 涉及自伤、暴力、医疗紧急时，先引导联系专业机构，不等待社群回应。
- 手机号仅管理员可见，家长之间互不可见。

## 4. 已知演示态与不一致

这些不是“新发现的 bug”，而是原型有意保留的演示态。修改前先判断：是补功能，还是更新文档口径。

| 位置 | 现状 | 建议 |
|---|---|---|
| 首页资源库 | 入口写“38 份资料”，五个文件夹合计为 20 份示例 | 统一为真实总数，或改成“5 个主题 · 长期留存” |
| 公告文案 | 公告里出现“在小程序里取消”，产品形态是 App | 统一为“App”或“应用内” |
| 取消报名 | 我的报名与报名成功页写“可以取消”，活动详情没有取消按钮 | 补齐取消入口、确认弹层、名额释放 |
| 活动档案 Tab | 未报名时原型显示锁定页，并注明真实产品会隐藏 | 正式版按说明隐藏；保留深链接兜底页 |
| 版本号 | 设置页写“无名树 v0.3 · 演示原型”，文档与仓库称 v3 | 统一版本命名规则 |
| 讨论区 | 三条示例回应为静态内容，输入框只弹 Toast | 接入真实楼层与发送状态 |
| 分享／文件 | 分享、文件打开只弹 Toast | 补齐分享卡片、文件预览与权限校验 |
| 审核 | “模拟管理员审核通过”按钮直接进入首页 | 替换为后台审核与结果通知 |
| 我的树洞 | 固定返回 p1/p4 两条，不是真实归属 | real data 接入时按作者过滤 |
| 报名手机号 | 写死只读掩码号码 | 从当前成员资料读取 |

产品说明文档第 11.4 节已把这些不一致写成评审项，不要在不改文档的情况下静默改掉。

## 5. 代码结构

`index.html` 是单文件，主要结构：

- `<style>`：设计变量、组件、页面布局、响应式、动画。
- `<script>`：`DATA`（活动／帖子／公告／文件夹／私信）、`S` 状态对象、`VIEWS` 路由表、各 `vXxx()` 视图函数、`render()`、`go/back/toast/overlay`、事件委托。
- 设备壳宽 392px，最高 848px；桌面端左侧是设计说明卡，窗口小于 480px 时切全屏手机视图。

### 5.0 工程代码（2026-11-08 起）

monorepo 三目录；工程约定的唯一来源是《技术架构设计》（`docs/无名树App技术架构设计-v1.0.md`），架构决策只写进该文档，其余文档引用不复制。

**server/（Go 1.23 + gin v1.10 + gorm v1.25，go.mod 已固定轻量版本，勿随手升级 gin）**

- 分层：module → service → repository → model；pkg 放零业务横切件（code/httpx/jwt/sms/audit/tc3/ratelimit/verifycode/crypto）。
- **云服务直连（2026-11-11）**：`pkg/tc3` 是手写腾讯云 TC3-HMAC-SHA256 签名客户端（不引 SDK，5s 超时，`BaseURL` 仅测试注入）；`sms.TencentSender` 走 SendSms v2021-01-11（+86 前缀，`SendStatusSet[0].Code=="Ok"` 才算成功，日志手机号只留后四位）；`audit.TencentAuditor` 走 TextModeration v2020-12-29，**危机词本地先行不外发**，本地词表先行双保险，Block→reject / Review→pending / Pass→allow，云异常降级 pending（先发后审）。配置：`TENCENT_SECRET_ID/KEY/REGION`、`SMS_SDK_APP_ID/SIGN_NAME/TEMPLATE_ID`；remote 缺凭证启动即失败。签名回归用官方文档示例的派生密钥链做固定向量（`tc3_test.go`）。
- 统一信封 `{"code":0,"message":"ok","data":...,"request_id"}`；16 个数字错误码与哨兵错误在 `internal/pkg/code`（唯一事实来源，改前先改文档）；**禁止裸 c.JSON**（CI envelope-guard 强制，本地也可跑同款 grep）。
- 已通端点：`/auth/sms-codes`、`/auth/login`（outcome 四路：member/application_pending/application_rejected/no_record）、`/auth/refresh`（access 7d + refresh 30d 一次性轮换，复用检测 → 2002 吊销全会话）；`/applications`（邀请码事务消费 + 手机号防重）；`/me`、`/me/badges`（仅布尔）、`/me/summary`、`PATCH /me`；`/admin/applications` 列表/通过/拒绝（事务建成员 + 审计日志）、`/admin/reject-reasons`（拒绝原因模板）、`/admin/members/:id/phone`（phone.view 逐次审计）。
- `cmd/seedadmin`：一次性创建管理员（`go run ./cmd/seedadmin -phone 138... -name 主管理员`，幂等；与正式准入同构加密）。
- 集成测试：`internal/it/it_test.go`（build tag `integration`）在真实 MySQL 上重放迁移并断言「申请 → 防重 → 审核 → 登录 → 资料/掩码/红点 → 权限拦截 → phone.view 审计」全链路；无 `WMT_IT_DSN` 自动跳过；CI 的 MySQL service 容器跑它（M1-14）。
- 手机号：member 主表不落手机号；密文在 member_secure（AES-256-GCM，主密钥 FILE_KEY），HMAC 摘要用于查找；明文只有管理端点解出。dev 无 FILE_KEY 时用固定派生键（config 内注明禁产）。
- 会话与频控在 Redis（不建表）；无 Redis 时自动退化为内存实现（仅 dev），验证码固定 123456（SMS_TYPE=mock）。
- 迁移：`migrations/000001_init.up.sql / .down.sql`（19 表）；迁移即事实，**禁止 AutoMigrate**；生产库有数据后脚本禁止 DROP/DELETE。
- 验证：`go build ./... && go vet ./... && go test ./...`（覆盖信封形状、错误码对齐、JWT 轮换、锁定语义、掩码等）。

**app/（Flutter，provider + go_router + dio + flutter_secure_storage）**

- `core/network/api_client.dart`：信封解包；2001 → 静默续期重放一次；2002 → 会话吊销回调。API 地址 `--dart-define=API_BASE_URL` 注入。
- `AuthController` 会话状态机（unknown/guest/pending/member）驱动 go_router `refreshListenable` 三态守卫；令牌存 flutter_secure_storage。
- 页面：`/gate` 邀请表单（含协议勾选）、`/gate/pending` 三步进度（禁返回）、`/login` 短信登录（60s 倒计时、按 outcome 分流）、`/me` 身份卡（含协议三入口）、`/me/settings`（昵称 8 字、四开关即时 PATCH、版本号 dart-define）、`/agreement/:doc` 协议静态页（user/privacy/rules，正文在 `features/agreement/agreement_content.dart`）。
- **协议页路由**：顶层 `/agreement/:doc`，guest 白名单放行；**不要挂到 `/gate` 子路由**——member 状态下 `/gate*` 会被 redirect 强制回 `/home`，我的页入口会失效。
- 本机无 Flutter SDK：`flutter analyze/test` 依赖 CI；平台目录初始化命令见 `app/README.md`。
- **Web 用户端**（2026-11-10 启用）：`kIsWeb` 时 API 基地址用相对 `/api/v1`；hash URL 策略（`flutter_web_plugins`，免疫 base-href 子路径）；宽屏 >900px 居中限宽 480px（避开 800×600 测试窗口）；CI 自动 `flutter build web` 并上传 web-dist artifact。踩坑：dart2js 无法精确表示 64 位整数字面量（token 生成改 Random.secure 字节串）；MaterialApp.builder 的 child 是 `Widget?`。

**miniprogram/（微信原生小程序，MP1）**

- `utils/api.js`：信封解包 + 2001 静默续期重放一次 + 2002 清会话回登录；令牌存 wx storage。
- 页面：gate（邀请表单+协议勾选）、pending（三步进度）、login（60s 倒计时、outcome 四路分流）、home（四入口+常驻风险提示+会话守卫）、hollow/list（心情筛选+下拉刷新+触底加载）、hollow/compose（匿名开关+client_token 幂等）、hollow/detail（危机求助卡+抱一下 toggle+楼层回应+wx.showActionSheet 举报）、hollow/myposts（长按删帖）、me（身份卡+开关 PATCH /me+协议入口+退出）、agreement（`?type=user|privacy|rules` 单页三篇，正文在 `pages/agreement/content.js`）。
- 红线静态检查进 `tools/check.js`：sitemap 全量 disallow、详情页 crisis 卡存在、首页风险提示存在、协议三篇已注册且 gate/my 入口存在。
- 上线前：BASE_URL 改正式域名并配置 request 合法域名；用户隐私保护指引后台提交。

**admin/（React 18 + Ant Design 5 + Vite 5，M1-6 最小版）**

- 页面：短信登录（校验 role=admin，非管理员拒绝）→ 申请列表（状态筛选 + 分页 + 通过/拒绝，拒绝原因下拉来自 `/admin/reject-reasons`）。
- `src/api.js` 信封解包；2001/2002 清 token 回登录；令牌存 localStorage（管理台内网使用，M1 口径）。
- dev：`npm run dev`（vite 把 `/api` 代理到 `127.0.0.1:8080`，无 CORS）；构建 `npm run build`（产物 base=/static/admin/，生产由 API 同源提供）。
- npm 安装间歇超时：npmmirror 与 npmjs 双源循环重试（本次 npmjs 第二源成功）。

### 5.1 小改

直接改 `index.html`，优先复用已有的视图函数、`card()`、`icon()`、`pill()`、`btn` 样式和色板变量。不要引入外部字体、CDN、图片或第二个 `<script>`。

### 5.2 大改

历史分片构建目录在当前机器已不存在（`build3` 未找到）。如要大改，建议重新拆分：

- `p1.html`：文档头 + 样式起点
- `base.css`：设计变量与组件
- `shell.html`：设备壳与左侧说明卡
- `app1.js`：DATA、S、工具函数、前段视图
- `app2.js`：后段视图、render、事件
- `tail.html`：闭合标签与脚本收尾

拼合后立刻做三件事：

1. `node --check all.js`，语法错会让整屏失效。
2. 用 vm + document 桩跑状态用例（至少覆盖 30 个状态）。
3. 截图前给 `out.html` 追加 `<style>` 禁用 `.rise` / `.floaty` 动画，否则首帧 `opacity:0` 会截出灰白屏；`icon()` 必须带 `width="16" height="16"`，否则 SVG 会按 300×150 撑爆。

### 5.3 不要做

- 不要为了“更活跃”加排行榜、热度、连续签到、公开动态。
- 不要把树洞改成聊天流或公开评论区。
- 不要把活动档案、孩子照片、老师手记放到公开页面。
- 不要提交真实手机号、孩子姓名、学校、住址或任何未脱敏素材。
- 不要在正式文档中使用 emoji。

## 6. 本地运行与检查

```powershell
# 直接打开
start .\index.html

# 或起一个本地服务
python -m http.server 8000
# 浏览器打开 http://localhost:8000/

# 服务端（本机 Go 在 D:\code\tools\go，未入 PATH；Git Bash）
export PATH=/d/code/tools/go/bin:$PATH
cd server && go build ./... && go vet ./... && go test ./...
# 无 DB/Redis 也能起（只挂 healthz，业务路由不加载）
go run ./cmd/api   # GET http://127.0.0.1:8080/api/v1/healthz
```

检查建议：

- 桌面宽度与 480px 以下各看一遍。
- 逐页点 8 条演示路径，确认没有空白页、破版、横向滚动、按钮被裁切。
- 重点看：活动详情的三 Tab、报名成功、档案锁定／解锁、树洞楼层、私信、设置。
- 改完 `index.html` 后，确认文件里只有一个 `<script>`，没有新增外链 `http(s)://` 依赖。

## 7. GitHub 发布

### 7.1 常规流程

```powershell
$repo = 'C:\Users\pigaw\Doubao\chats\2026-11-07\new-chat\wumingtree-app'
& 'C:\Program Files\Git\cmd\git.exe' -C $repo status
& 'C:\Program Files\Git\cmd\git.exe' -C $repo add .
& 'C:\Program Files\Git\cmd\git.exe' -C $repo commit -m "docs: update product spec"
& 'C:\Program Files\Git\cmd\git.exe' -C $repo push origin main
```

当前机器上 `git.exe` 不在 PATH，需用完整路径 `C:\Program Files\Git\cmd\git.exe`。Windows 凭据管理器里有 `git:https://github.com` 记录；如凭据失效，先让用户登录，不要自行创建令牌或写入仓库。

### 7.2 git push 被网络阻断时

历史经验：`github.com:443` 可能超时，但 `api.github.com` 可达。此时不要反复重试 push，改用 GitHub Contents API：

```text
PUT /repos/zhumin2/wumingtree-app/contents/<path>
body: { message, content(base64), branch: "main", sha? }
```

要点：

- 新增文件不传 `sha`；更新已有文件必须先取当前 `sha`。
- JSON 用 UTF-8 无 BOM 写入临时文件，再 `gh api --input`，避免中文乱码。
- 二进制文件（docx、png）同样用 base64 上传。
- 上传后用远端 `sha`、Pages 构建状态、线上 URL 三重核验。

**2026-11-08 实战记录（本次推送即走此通道）**：`github.com:443` 完全阻断（DNS 绕行与 `ssh.github.com:443` 均不通），仅 `api.github.com` 与 `codeload` 可达。用 Git Data API 复刻提交：逐 blob 上传（base64）→ 以 `base_tree` 建 tree → `POST /git/commits`（message/author/committer 原样）→ `PATCH /git/refs/heads/main`。**远端 tree 与本地逐提交一致（内容字节级相同），但 commit SHA 与本地不同**：GitHub 对带偏移的 ISO 日期按「壁钟即 UTC」处理，并剥掉 message 尾部换行，提交对象字节与本地必然有差，不必强求一致。

下次 git 直连恢复后做一次无损对齐（远端内容已验证与本地一致）：

```powershell
git fetch origin
git reset --hard origin/main
```

在这之前若再次推送：优先重试 `git push`（间歇性），仍不通则继续用 Git Data API 追加提交；本地新增提交与远端链会持续并存，最终以一次 `fetch + reset --hard` 收敛。

**当前状态（2026-11-11 更新）**：远端与本地已收敛——本机从远端全新克隆（`git clone` 直连正常），本地 main = 远端 main，均为含 M2 + MP1 + Web 的线性历史。7.2 的 API 回退通道保留作应急（`github.com:443` 阻断时）；git 直连正常时直接 `git push origin main`，**不要再基于旧 SHA 链 rebase**。历史分叉提交（旧本地 `fd7ebac` 等）已随新 clone 消解，无需再 `reset --hard`。

当前 Windows 机器没有可直接使用的 `gh` 命令；如 API 回退，需要先安装 gh 或改用现有凭据的 git 推送（本次即用 `git credential fill` 取凭据直调 REST）。不要凭记忆假设 `gh` 一定存在。

### 7.3 分叉处理

如果本地 clone 与远端分叉，先 `git fetch`、看 `git log --oneline --graph`，再决定合并或重建提交。**不要 `push --force`**，除非用户明确要求且确认线上内容可覆盖。

## 8. 产品说明文档

文件：`docs/无名树App产品说明文档-v1.0.docx`

- 依据 v3 原型整理，12 章：文档说明、产品定位、用户与场景、产品原则、信息架构、核心功能、关键流程、数据模型、视觉与交互、非功能与合规、原型边界、开放问题。
- 34 张表、1 张首页截图；Word 排版为 A4、2.5cm 页边距、宋体正文、黑体标题、动态目录。
- 修改 Word 前必须重新阅读 `word` skill 的 `SKILL.md`，不要凭记忆直接改。
- 文档中的“已知不一致”是刻意披露，不是错误；如原型已修复，同步更新第 11.4 节。
- 正式文档不要加 emoji，不要把演示计数写成真实运营数据。

## 9. 接手建议顺序

1. 读本文件。
2. 打开 `index.html`，按第 3.2 节走一遍 8 条路径。
3. 读产品说明文档第 2、4、6、11 章，建立产品判断。
4. 选一个明确的小任务，不要一上来重做视觉。
5. 改完先本地检查，再提交；提交信息写清楚“改了什么、为什么”。

## 9.0 端矩阵（2026-11-10）

| 端 | 目录 | 使用者 | 分发 | 进度 |
|---|---|---|---|---|
| Flutter App | `app/` | 家长（主力） | Android APK（apkserver）；iOS 待账号 | M0–M2 ✓ |
| 用户端 Web | `app/`（同一代码库） | 家长（浏览器） | `flutter build web` → `WEB_DIR` → API 同源 `/static/web/`（no-cache） | 已启用 ✓ |
| 微信小程序 | `miniprogram/` | 家长（轻量入口） | 微信平台（上线流程见 9.1） | MP1 ✓ |
| 管理台 Web | `admin/` | 管理员/带队老师 | 同源 `/static/admin/` | 审核+举报 ✓ |

服务端 `server/` 是唯一数据入口，四端共用同一 API 契约与信封约定；M3+ 新功能服务端先行、多端跟进。

## 9.1 小程序上线路线（MP1 → 发布；关键路径是备案，合计约 4–8 周）

| 步骤 | 事项 | 说明 | 谁办 |
|---|---|---|---|
| 1 | 注册小程序账号（mp.weixin.qq.com） | **必须企业/个体户主体**——树洞是 UGC，「社交-社区/论坛」类目不接受个人 | 用户 |
| 2 | 小程序认证（300 元/年）+ 类目「社交-社区/论坛」 | 需域名 ICP 备案材料 | 用户 |
| 3 | 域名 ICP 备案（2–4 周）→ 小程序备案（1–4 周） | 微信平台内提交 | 用户 |
| 4 | 云主机 + 域名 + HTTPS + `install.sh` 部署 + `seedadmin` | M0-5 清单 | 用户+agent |
| 5 | `SMS_TYPE=remote` / `AUDIT_MODE=remote` 真实接入 | **代码侧已完成（2026-11-11）**：腾讯云直连（`server/internal/pkg/tc3` 手写 TC3-HMAC-SHA256 + `sms.TencentSender` + `audit.TencentAuditor`）；生产只需在 env 填 `TENCENT_SECRET_ID/KEY`、`SMS_SDK_APP_ID/SIGN_NAME/TEMPLATE_ID` 并切 remote | agent |
| 6 | 用户协议/隐私政策/社区规范页面 + gate 与我的页入口 | **代码侧已完成（2026-11-11）**——审核必查项：正文正本 `docs/agreements/*.md`；Flutter/Web `/agreement/:doc` + gate 勾选 + 我的页三入口；小程序 `pages/agreement`（单页 ?type=）+ gate 勾选 + 我的页「关于」分组；`tools/check.js` 已加静态红线检查 | agent |
| 7 | `utils/config.js` BASE_URL 改正式域名；AppID 替换 `touristappid` | 正式域名需 HTTPS+备案 | agent |
| 8 | DevTools 上传 → 体验版 → 真机内测全链路 | mock 短信 123456 → 切真实短信 | 用户+agent |
| 9 | 提交审核 | 材料：类目证明（ICP 截图）、测试账号（邀请码+说明）、隐私保护指引（声明手机号/昵称/storage）、UGC 安全机制说明（机审+人工+举报已具备） | 用户 |
| 10 | 发布；上线 30 日内公安联网备案（beian.mps.gov.cn）+ 页脚展示 | | 用户 |

## 9.2 优先候选任务（对应开发计划，按序）

- ~~上线代码侧第 5、6 项（短信/内容安全真实接入 + 协议页）~~ **已完成 2026-11-11**；剩余为用户线下办理（短信签名模板备案、ICP/小程序备案）后填 env 切 remote。
- M1-13 垂直切片联调：需要真实环境（install.sh 装机 + seedadmin 建管理员），mock 短信走通「申请 → 审核（管理台）→ 登录 → 树洞发帖」全链路。
- M3 公告与资源库（服务端横向复制 M1/M2 模式；文件存储用本地盘 + 鉴权代理；小程序端同步新增公告/资源页）。
- 小程序端 MP2：wx.login/code2Session 身份打通（需 AppID 与企业资质）、订阅消息通知、分享开关评估。
- 长周期事项跟进（用户线下办理）：ICP/App 备案、短信签名模板、厂商推送账号、iOS 开发者账号。
- M4 活动与档案（含报名事务与 audience 过滤）随后启动。

## 10. 环境备注

- 当前 Windows 环境：PIG2LAPTOP。Git Bash 会话内 `git` 已在 PATH；`node` v24 可用；`python` 可用；本机已装 Go（`go version` = go1.27.3，直接在 PATH，可直接跑 server 的 build/vet/test）；**无 Flutter SDK**，客户端验证依赖 CI。
- **gofmt 假阳性**：本机 `core.autocrlf=true`，`gofmt -l` 会把几乎全仓（含未改动文件）报成未格式化，实际 diff 只有 `^M`（CRLF）。**不要 `gofmt -w ./...`**——会把整仓变成噪音 diff；只 format 新建文件，且以 CI（Linux/LF）为准。
- Go 模块代理间歇超时是常态：GOPROXY 在 `goproxy.cn` 与 `proxy.golang.org` 之间循环重试即可（同参考架构踩坑 14）。
- git 身份已在仓库级配置为 zhumin2 <37998940+zhumin2@users.noreply.github.com>（与历史提交一致）。
- 云电脑环境：Linux，路径为 `/home/user/Doubao/chats/...`；**不能读取 Windows 的 `D:\` 或 `C:\Users\...`**。跨环境操作先确认文件在哪一侧。
- 仓库是唯一事实来源；本地旧目录可能不存在或为空，不要据此判断项目丢失。
- 不要把 token、cookie、真实用户信息写进仓库、README 或本文件。

## 11. 变更记录

| 日期 | 变更 |
|---|---|
| 2026-11-07 | v3 明亮活泼版原型完成；README、preview 更新；产品说明文档 v1.0 完成；本接手说明建立 |
| 2026-11-08 | 文档定稿：技术设计 / 架构设计 / 细化设计（DDL 19 表 + OpenAPI 契约）/ 开发计划（M0–M6，111 人天）。M0 完成：monorepo 骨架（app/server/admin）、部署三件套（install.sh / systemd / wumingtree.sh）、CI（含信封铁律 guard）。M1 服务端完成：迁移 19 表、统一信封与 16 个数字错误码、短信验证码登录（outcome 四路）、邀请码事务消费与防重、刷新令牌轮换与复用检测、我的模块（掩码手机号）、审核端点与 phone.view 审计；服务层/组件 20+ 用例全绿（build/vet/test）。M1 客户端完成：网络层 2001 续期 / 2002 吊销钩子、三态路由守卫、邀请 / 登录 / 我的 / 设置四页；待 CI（无本地 Flutter SDK）。M1 剩余：管理台 UI、设计系统组件、CI 集成测试、垂直切片联调 |
| 2026-11-08 | 推送上远端：`github.com` 直连被阻断，5 个提交经 Git Data API 复刻（远端 tree 与本地全 MATCH，SHA 因 GitHub 日期/换行归一而不同）；本地对齐命令见 7.2。远端 main 头：`08832c6` 之后另有 agents.md 补记提交 |
| 2026-11-08 | M1-6 完成：管理台最小版（React+AntD，短信登录/申请列表/通过/拒绝+原因模板端点）；M1-14 完成：CI 加 MySQL service，集成测试在真库重放迁移并断言准入全链路（权限矩阵关键格/4203/3002/掩码/红点布尔/phone.view 审计）；配套 `cmd/seedadmin` 幂等建管理员。本地 build/vet/test 全绿，管理台 `npm run build` 通过 |
| 2026-11-08 | CI 五轮迭代后**全绿**（run5 头 `86054d6`：server build/vet/test + 真库集成测试 + envelope guard；client analyze 零告警 + test）。迭代修掉的真问题：测试邀请码 13 位不符合 12 位格式校验、集成测试编排与短信 60s 频控冲突（login 助手改为按手机号缓存令牌、pending 断言改用一次性手机号）、审核需按提交返回的申请 ID 精确批准（列表倒序 items[0] 非目标）、Dart：Envelope 重复构造器/go_route const/弃用 activeColor/smoke test 用 const Color 比较 |
| 2026-11-08 | **M2 树洞完成**：服务端（audit 机审钩子 mock 词表+危机词不拦截、idempot 幂等存储 Acquire/Save/Load/Release、cursor 游标、树洞四域仓库与服务、举报 reporter json:- 永不序列化、badges Redis/Memory）+ 客户端（M1-10 组件 ThickButton/Sticker/WmtPill/LockedCard/MoodChip/ErrorRetry、树洞列表/发帖/详情/我的树洞四页、路由接线）。CI 七轮迭代后全绿（头 `84aad4e`） |
| 2026-11-08 | **M2 踩坑沉淀**：gorm 对带主键的匿名嵌入结构按 has-one 关联解析（embedded 标签也无效）→ 行结构改扁平字段；写操作幂等需在业务失败时 Release 槽位；集成测试编排要与短信 60s 频控解耦（按手机号缓存令牌、一次性手机号做 pending 断言）；审核按提交返回的申请 ID 精确批准；async gap 的 BuildContext 需 await 前捕获引用 |
| 2026-11-08 | **Git 状态**：远端 main 头 `86054d6`（CI 全绿）；本地 main `1eba480`，两者内容一致（tree 同源、SHA 因 GitHub 日期/换行归一而不同）。本地曾因一次误操作 reset 回退到 8eb99e5，已从 reflog 恢复到含全部修复的 1eba480。**git 直连恢复后执行 `git fetch origin && git reset --hard origin/main` 收敛；在此之前不要基于本地旧历史发起新提交推送，先用 API 通道（7.2）** |
| 2026-11-10 | **小程序端 MP1 完成**（miniprogram/，原生微信小程序）：9 页面（邀请/审核/登录/首页/树洞列表/发帖/详情含危机卡/我的树洞/我的含设置）；utils 信封层（2001 续期/2002 吊销）与设计 token 对齐 Flutter 端；sitemap 全量 disallow + 红线静态检查进 tools/check.js + CI 新增 miniprogram job（JS 语法 + 结构校验）。登录沿用手机号+验证码；wx.login 打通列 MP2。结构校验 9 页全过、JS 语法全过 |
| 2026-11-10 | **用户端 Web 启用**（参考架构 5.2 模式）：Flutter 同一代码库出 Web 形态——kIsWeb 相对 API 路径、hash URL 策略、宽屏居中限宽；server 新增 WEB_DIR 静态托管 `/static/web/`（no-cache 铁律）+ env/update 接线；CI client job 增加 `flutter build web` + web-dist artifact。踩坑：dart2js 64 位字面量（token 改 Random.secure）、MaterialApp.builder child 可空。CI 全绿（头 `cd87f10`） |
| 2026-11-11 | **四端矩阵与小程序上线路线固化进本文件**（9.0/9.1）；Git 状态更新（远端 9dddda6 CI 全绿：M2+MP1+Web；本地 fd7ebac 内容一致）。上线代码侧待办明确：RemoteSender/RemoteAuditor 真实接入（推荐腾讯云）、协议/隐私页、BASE_URL 与 AppID 替换；关键路径为 ICP+小程序备案（4–8 周） |
| 2026-11-11 | **上线代码侧第 5、6 项完成**。①腾讯云直连：新增 `pkg/tc3`（手写 TC3-HMAC-SHA256，纯标准库，官方文档固定向量测试）+ `sms.TencentSender`（SendSms v2021-01-11）+ `audit.TencentAuditor`（TextModeration v2020-12-29，危机词本地先行不外发、本地词表先行、云异常降级 pending）；config 增 TENCENT_* / SMS_SDK_APP_ID/SIGN_NAME/TEMPLATE_ID，remote 缺凭证启动即失败；env.example 同步。②协议三件套：正本 `docs/agreements/*.md`；Flutter/Web 顶层 `/agreement/:doc`（guest 白名单已放行，不挂 /gate 子路由因 member 会被 redirect 回 /home）+ gate 勾选 + 我的页三入口 + widget test；小程序 `pages/agreement` 单页（?type=）+ gate 勾选 + 我的页「关于」分组 + check.js 静态红线。服务端 build/vet/test 全绿；miniprogram 结构校验 10 页通过；Flutter 依赖 CI |
| 2026-11-11 | **本地仓库重建**：`git clone` 直连成功（github.com:443 已恢复），本地与远端 history 收敛，旧分叉提交不再需要；7.2 的 API 通道保留为应急 |
