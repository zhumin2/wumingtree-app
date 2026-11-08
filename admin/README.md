# admin · 管理台（React + Vite + Ant Design）

管理台 Web 端：申请审核、公告发布、活动与档案管理、举报处理、成员管理。

端点与功能范围见《无名树App技术设计文档-v1.0》第 4.4 节管理端点表；
构建产物随仓库分发到 `server/deploy/linux/admin/`，由 Go API 同源提供
（`/static/admin/`），认证复用 JWT（role=admin）。

当前状态：M1-6（管理台最小版：登录 + 申请列表 + 通过/拒绝）时初始化工程。
