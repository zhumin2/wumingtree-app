#!/usr/bin/env bash
# =====================================================================
# install.sh — 无名树 API 单机安装（部署三件套之一，架构设计 5.1）
# 适用：CentOS/RHEL(yum) 与 Debian/Ubuntu(apt)，root 运行。
# 步骤：系统依赖 → 建库建用户 → 目录 → 迁移（可重入）→ systemd → 管理脚本 → env 模板。
# 红线：检测到业务表已有数据时跳过建表段，保证可重复执行不炸（架构设计 3.7）。
# =====================================================================
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
SRC_DIR=$(cd "$SCRIPT_DIR/../.." && pwd)          # server/
REPO_DIR=$(cd "$SRC_DIR/.." && pwd)               # 仓库根
OPT=/opt/wumingtree
DB_NAME=wumingtree
DB_USER=wumingtree
DB_PASS=${DB_PASS:-$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')}
SERVICE=wumingtree-api

say() { echo "[install] $*"; }
die() { echo "[install][ERROR] $*" >&2; exit 1; }

# ---------- 1. 系统依赖 ----------
PKG=()
command -v curl   >/dev/null 2>&1 || PKG+=(curl)
command -v tar    >/dev/null 2>&1 || PKG+=(tar)
command -v gzip   >/dev/null 2>&1 || PKG+=(gzip)
if command -v yum >/dev/null 2>&1; then
  PKG+=(mysql-server redis)
  say "检测到 yum，安装系统依赖: ${PKG[*]}"
  yum install -y "${PKG[@]}" || die "yum 安装失败，请手动安装 MySQL 与 Redis"
  systemctl enable --now mysqld redis || systemctl enable --now mysql redis
elif command -v apt-get >/dev/null 2>&1; then
  PKG+=(default-mysql-server redis-server)
  say "检测到 apt，安装系统依赖: ${PKG[*]}"
  apt-get update -y && apt-get install -y "${PKG[@]}" || die "apt 安装失败，请手动安装 MySQL 与 Redis"
  systemctl enable --now mysql redis-server 2>/dev/null || systemctl enable --now mariadb redis-server
else
  die "未识别的包管理器（仅支持 yum/apt），请手动安装 MySQL 8 与 Redis 后重跑"
fi

# ---------- 2. 建库建用户（幂等） ----------
say "创建数据库与账号"
MYSQL="mysql"
if command -v mysql >/dev/null 2>&1; then MYSQL=mysql; fi
$MYSQL -Nbe "CREATE DATABASE IF NOT EXISTS \`$DB_NAME\` DEFAULT CHARSET utf8mb4;"
$MYSQL -Nbe "CREATE USER IF NOT EXISTS '$DB_USER'@'localhost' IDENTIFIED BY '$DB_PASS';"
$MYSQL -Nbe "ALTER USER '$DB_USER'@'localhost' IDENTIFIED BY '$DB_PASS';"
$MYSQL -Nbe "GRANT ALL PRIVILEGES ON \`$DB_NAME\`.* TO '$DB_USER'@'localhost'; FLUSH PRIVILEGES;"

# ---------- 3. 目录与运行用户 ----------
say "创建目录与运行用户"
id wmt >/dev/null 2>&1 || useradd --system --home-dir /nonexistent --shell /usr/sbin/nologin wmt
mkdir -p "$OPT/bin" "$OPT/uploads" "$OPT/backups" "$OPT/src"
cp -r "$REPO_DIR/." "$OPT/src/" 2>/dev/null || die "复制仓库到 $OPT/src 失败"
chown -R wmt:wmt "$OPT/uploads"

# ---------- 4. 迁移（可重入：业务表有数据则跳过建表段） ----------
say "执行数据库迁移"
INIT_SQL="$SRC_DIR/migrations/000001_init.up.sql"
[ -f "$INIT_SQL" ] || die "缺少初始迁移文件: $INIT_SQL"
HAS_MEMBER=$($MYSQL -Nbe "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='$DB_NAME' AND table_name='member';")
if [ "$HAS_MEMBER" = "1" ]; then
  HAS_DATA=$($MYSQL -Nbe "SELECT COUNT(*) FROM \`$DB_NAME\`.member;")
  if [ "$HAS_DATA" != "0" ]; then
    say "member 表已存在且有数据，跳过初始迁移（可重入保护）"
  else
    $MYSQL "$DB_NAME" < "$INIT_SQL"
    say "初始迁移已应用（member 表存在但为空）"
  fi
else
  $MYSQL "$DB_NAME" < "$INIT_SQL"
  say "初始迁移已应用"
fi

# ---------- 5. 构建或取预编译产物 ----------
say "准备 API 二进制"
if command -v go >/dev/null 2>&1; then
  (cd "$SRC_DIR" && GOPROXY=${GOPROXY:-https://goproxy.cn,direct} CGO_ENABLED=0 go build -o "$OPT/bin/api" ./cmd/api) \
    || die "编译失败"
  say "本机编译完成"
else
  PREBUILT="$SRC_DIR/deploy/linux/bin/api"
  if [ -f "$PREBUILT" ]; then
    cp "$PREBUILT" "$OPT/bin/api"
    say "使用预编译产物"
  else
    say "本机无 Go 且无预编译产物，跳过二进制（可用 wumingtree.sh update 在有 Go 的环境补齐）"
  fi
fi
chmod +x "$OPT/bin/api" 2>/dev/null || true

# ---------- 6. env 文件（不存在才生成，避免覆盖真实密钥） ----------
if [ ! -f "$OPT/wumingtree.env" ]; then
  sed -e "s|^DB_DSN=.*|DB_DSN=$DB_USER:$DB_PASS@tcp(127.0.0.1:3306)/$DB_NAME?parseTime=true\&loc=UTC|" \
      "$SRC_DIR/deploy/linux/wumingtree.env.example" > "$OPT/wumingtree.env"
  say "已生成 $OPT/wumingtree.env（生产部署前必须改 JWT_SECRET / FILE_KEY / ENV）"
else
  say "env 已存在，保持不变"
fi
chown root:wmt "$OPT/wumingtree.env" && chmod 640 "$OPT/wumingtree.env"

# ---------- 7. systemd 单元与管理脚本 ----------
say "安装 systemd 单元与管理脚本"
cp "$SRC_DIR/deploy/linux/wumingtree-api.service" /etc/systemd/system/$SERVICE.service
cp "$SRC_DIR/deploy/linux/wumingtree.sh" /usr/local/bin/wumingtree.sh && chmod +x /usr/local/bin/wumingtree.sh
systemctl daemon-reload
systemctl enable $SERVICE

say "完成。下一步："
say "  1) 编辑 $OPT/wumingtree.env（生产必改 JWT_SECRET / FILE_KEY，openssl rand -hex 32）"
say "  2) systemctl start $SERVICE && wumingtree.sh health"
say "  3) crontab 增加每日备份: 30 3 * * * /usr/local/bin/wumingtree.sh backup"
