#!/usr/bin/env bash
# =====================================================================
# wumingtree.sh — 无名树 API 管理脚本（部署三件套之一，架构设计 5.1）
# 用法：wumingtree.sh start|stop|restart|status|health|logs|backup|update
# 约定：直接以 root 运行（sudo 的 secure_path 通常不含 /usr/local/bin，别加 sudo）。
# =====================================================================
set -euo pipefail

APP=wumingtree-api
OPT=/opt/wumingtree
REPO=${REPO:-$OPT/src}
BIN=$OPT/bin/api
ENV_FILE=$OPT/wumingtree.env
HEALTH_URL=${HEALTH_URL:-http://127.0.0.1:8080/api/v1/healthz}
BACKUP_DIR=$OPT/backups
BACKUP_KEEP=7

say() { echo "[wumingtree] $*"; }
die() { echo "[wumingtree][ERROR] $*" >&2; exit 1; }

svc() { systemctl "$@" "$APP"; }

load_dsn() {
  [ -f "$ENV_FILE" ] || die "缺少 $ENV_FILE"
  DB_DSN=$(grep -E '^DB_DSN=' "$ENV_FILE" | head -1 | cut -d= -f2-)
  [ -n "$DB_DSN" ] || die "ENV 文件缺少 DB_DSN"
}

cmd_start()   { svc start; say started; }
cmd_stop()    { svc stop;  say stopped; }
cmd_restart() { svc restart; say restarted; }

cmd_status() {
  svc status || true
}

cmd_health() {
  # 探活：最多 15 次 x2s，成功输出组件状态
  for i in $(seq 1 15); do
    if body=$(curl -fsS --max-time 3 "$HEALTH_URL" 2>/dev/null); then
      say "healthy: $body"
      return 0
    fi
    sleep 2
  done
  die "health check failed: $HEALTH_URL"
}

cmd_logs() {
  journalctl -u "$APP" -n 200 -f
}

cmd_backup() {
  load_dsn
  local ts user pass dbname f
  ts=$(date +%F_%H%M%S)
  mkdir -p "$BACKUP_DIR"
  # DSN 形如 user:pass@tcp(host:port)/dbname?params（正则须入变量，括号才能安全解析）
  local dsn_re='^([^:]+):([^@]*)@tcp\([^)]+\)/([A-Za-z0-9_]+)'
  if [[ $DB_DSN =~ $dsn_re ]]; then
    user=${BASH_REMATCH[1]}; pass=${BASH_REMATCH[2]}; dbname=${BASH_REMATCH[3]}
  else
    die "DB_DSN 无法解析，无法备份"
  fi
  f="$BACKUP_DIR/db-$ts.sql.gz"
  mysqldump -u"$user" -p"$pass" --single-transaction --routines "$dbname" | gzip > "$f"
  say "数据库已备份: $f"
  # 上传目录（档案照片是唯一真源）
  tar -czf "$BACKUP_DIR/uploads-$ts.tar.gz" -C "$OPT" uploads 2>/dev/null || say "uploads 为空，跳过归档"
  say "上传目录已备份: $BACKUP_DIR/uploads-$ts.tar.gz"
  # 保留最近 BACKUP_KEEP 份
  ls -1t "$BACKUP_DIR"/db-*.sql.gz 2>/dev/null | tail -n +$((BACKUP_KEEP + 1)) | xargs -r rm -f
  ls -1t "$BACKUP_DIR"/uploads-*.tar.gz 2>/dev/null | tail -n +$((BACKUP_KEEP + 1)) | xargs -r rm -f
  say "旧备份清理完成（保留 $BACKUP_KEEP 份）"
}

cmd_update() {
  [ -d "$REPO/.git" ] || die "仓库目录不存在: $REPO（REPO=/path/to/wumingtree-app $0 update）"
  cd "$REPO"
  # 仓库混入本地改动会让 pull 失败：先 stash（参考架构踩坑 15）
  if [ -n "$(git status --porcelain)" ]; then
    say "检测到本地改动，先 stash"
    git stash -u >/dev/null
  fi
  say "拉取最新代码"
  git fetch origin
  git reset --hard origin/main
  mkdir -p "$OPT/bin"

  # 优先本机编译（GOPROXY 默认 goproxy.cn）；失败回退仓库内预编译产物
  if command -v go >/dev/null 2>&1; then
    say "本机编译 API"
    (cd "$REPO/server" && GOPROXY=${GOPROXY:-https://goproxy.cn,direct} CGO_ENABLED=0 go build -o "$BIN.new" ./cmd/api) \
      || say "编译失败，回退预编译产物"
  fi
  if [ ! -f "${BIN}.new" ]; then
    PREBUILT="$REPO/server/deploy/linux/bin/api"
    [ -f "$PREBUILT" ] || die "无预编译产物且本机无 Go，无法更新"
    cp "$PREBUILT" "$BIN.new"
  fi
  # 同分区 mv 原子替换，再重启
  mv -f "$BIN.new" "$BIN"
  # 用户端 Web 产物随仓库走（参考架构 5.2）
  if [ -d "$REPO/server/deploy/linux/web" ]; then
    mkdir -p "$OPT/web"
    cp -r "$REPO/server/deploy/linux/web/." "$OPT/web/"
    say "用户端 Web 产物已更新"
  fi
  svc restart
  cmd_health
  say "update 完成"
}

case "${1:-status}" in
  start)   cmd_start ;;
  stop)    cmd_stop ;;
  restart) cmd_restart ;;
  status)  cmd_status ;;
  health)  cmd_health ;;
  logs)    cmd_logs ;;
  backup)  cmd_backup ;;
  update)  cmd_update ;;
  *) die "用法: $0 start|stop|restart|status|health|logs|backup|update" ;;
esac
