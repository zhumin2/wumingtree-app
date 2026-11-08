-- =====================================================================
-- 无名树 App · 数据库初始化 DDL v1.1
-- 依据：《无名树App技术架构设计-v1.0》三、服务端架构
--       + 《无名树App技术设计文档-v1.0》第 4.2 节 + 《无名树App细化设计-v1.0》
-- v1.1 变更（对齐参考架构「拾光 ShinePick」模式）：
--   1) 手机号密文拆分到 member_secure 独立表，member 主表不落敏感字段；
--   2) 删除 sms_verify 表，验证码改存 Redis TTL 5 分钟（架构设计 3.3）；
--   3) oss_key 更名 file_key（存储为本地盘鉴权代理，OSS 为触模扩展项）；
--   4) 版本表由 golang-migrate 自动维护（schema_migrations），不在此建表。
-- 目标：MySQL 8.0.16+（依赖 CHECK 约束）；字符集 utf8mb4
-- 约定：时间列 DATETIME(3)，统一 UTC 存储（由应用层保证）；
--       主键 BIGINT UNSIGNED AUTO_INCREMENT；软删用 status 或 deleted_at。
-- 使用：迁入 server 仓库 migrations/000001_init.up.sql（配套 .down.sql 成对）。
-- 红线：生产库有真实数据后，一切脚本禁止 DROP/DELETE。
-- =====================================================================
SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 1. member 成员（家长 / 带队老师 / 管理员统一建模，角色区分；主表无敏感字段）
-- ---------------------------------------------------------------------
CREATE TABLE member (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  nickname      VARCHAR(8)      NOT NULL COMMENT '昵称，最多 8 字（产品规则）',
  role          VARCHAR(16)     NOT NULL DEFAULT 'member' COMMENT 'member/teacher/admin',
  status        VARCHAR(16)     NOT NULL DEFAULT 'active' COMMENT 'active/disabled',
  default_anon  TINYINT(1)      NOT NULL DEFAULT 1 COMMENT '默认匿名发帖',
  dnd           TINYINT(1)      NOT NULL DEFAULT 1 COMMENT '免打扰 22:00-08:00',
  notify_notice TINYINT(1)      NOT NULL DEFAULT 1 COMMENT '公告提醒',
  notify_hollow TINYINT(1)      NOT NULL DEFAULT 1 COMMENT '树洞回应提醒',
  joined_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '加入时间',
  deleted_at    DATETIME(3)     NULL COMMENT '注销时间（软删，成员不物理删除）',
  created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_member_role_status (role, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='成员';

-- ---------------------------------------------------------------------
-- 2. member_secure 手机号隔离存储（架构设计 3.3）
--    密文 AES-256-GCM，主密钥来自部署 env 的 FILE_KEY；摘要用于登录查找与防重
-- ---------------------------------------------------------------------
CREATE TABLE member_secure (
  member_id    BIGINT UNSIGNED NOT NULL,
  phone_cipher VARBINARY(512)  NOT NULL COMMENT '手机号密文',
  phone_hash   CHAR(64)        NOT NULL COMMENT 'HMAC-SHA256 摘要，不可逆',
  updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (member_id),
  UNIQUE KEY uk_secure_phone_hash (phone_hash),
  CONSTRAINT fk_secure_member FOREIGN KEY (member_id) REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='手机号隔离存储';

-- ---------------------------------------------------------------------
-- 3. invite_code 邀请码
-- ---------------------------------------------------------------------
CREATE TABLE invite_code (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code        VARCHAR(32)     NOT NULL COMMENT '随机 12 位，展示格式 WMT-XXXX-XXXX',
  status      VARCHAR(16)     NOT NULL DEFAULT 'active' COMMENT 'active/disabled/exhausted',
  valid_from  DATETIME(3)     NOT NULL,
  valid_until DATETIME(3)     NOT NULL,
  max_uses    INT             NOT NULL DEFAULT 1 COMMENT '默认单次有效',
  used_count  INT             NOT NULL DEFAULT 0,
  created_by  BIGINT UNSIGNED NOT NULL COMMENT '创建管理员',
  created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_invite_code (code),
  KEY idx_invite_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='邀请码';

-- ---------------------------------------------------------------------
-- 4. application 加入申请
--    防重在应用层：同 phone_hash 存在 submitted 申请或 active 成员即拒绝
--    （历史 rejected 允许再次申请，故不做 DB 唯一约束）
-- ---------------------------------------------------------------------
CREATE TABLE application (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  invite_code_id BIGINT UNSIGNED NOT NULL,
  nickname       VARCHAR(8)      NOT NULL,
  phone_cipher   VARBINARY(512)  NOT NULL COMMENT '申请阶段手机号密文（审核联系用）',
  phone_hash     CHAR(64)        NOT NULL,
  status         VARCHAR(16)     NOT NULL DEFAULT 'submitted' COMMENT 'submitted/approved/rejected',
  reject_reason  VARCHAR(200)    NULL COMMENT '拒绝原因（模板文案）',
  reviewed_by    BIGINT UNSIGNED NULL COMMENT '审核管理员',
  reviewed_at    DATETIME(3)     NULL,
  member_id      BIGINT UNSIGNED NULL COMMENT '通过后生成的成员',
  created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_app_status (status, created_at),
  KEY idx_app_phone (phone_hash),
  CONSTRAINT fk_app_invite  FOREIGN KEY (invite_code_id) REFERENCES invite_code (id),
  CONSTRAINT fk_app_member  FOREIGN KEY (member_id)      REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='加入申请';

-- ---------------------------------------------------------------------
-- 5. notice 公告（单向通知，无评论能力——红线）
-- ---------------------------------------------------------------------
CREATE TABLE notice (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  pinned       TINYINT(1)      NOT NULL DEFAULT 0,
  title        VARCHAR(50)     NOT NULL,
  segments     JSON            NOT NULL COMMENT '分段正文，如 [{"text":"..."}]',
  published_by BIGINT UNSIGNED NOT NULL,
  published_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_notice_list (pinned DESC, published_at DESC),
  CONSTRAINT fk_notice_publisher FOREIGN KEY (published_by) REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='公告';

-- ---------------------------------------------------------------------
-- 6. notice_read 公告个人已读（只做个人状态，不做聚合计数——红线）
-- ---------------------------------------------------------------------
CREATE TABLE notice_read (
  member_id BIGINT UNSIGNED NOT NULL,
  notice_id BIGINT UNSIGNED NOT NULL,
  read_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (member_id, notice_id),
  KEY idx_notice_read_notice (notice_id),
  CONSTRAINT fk_nr_member FOREIGN KEY (member_id) REFERENCES member (id),
  CONSTRAINT fk_nr_notice FOREIGN KEY (notice_id) REFERENCES notice (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='公告个人已读';

-- ---------------------------------------------------------------------
-- 7. post 树洞帖（author_id 始终保留——删除权与治理；匿名时 API 不回传——红线）
-- ---------------------------------------------------------------------
CREATE TABLE post (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  author_id      BIGINT UNSIGNED NOT NULL,
  mood           VARCHAR(8)      NOT NULL COMMENT 'share/tired/ask/talk',
  title          VARCHAR(50)     NOT NULL,
  body           TEXT            NULL,
  is_anonymous   TINYINT(1)      NOT NULL DEFAULT 1,
  status         VARCHAR(16)     NOT NULL DEFAULT 'published' COMMENT 'published/removed',
  removed_reason VARCHAR(200)    NULL COMMENT 'by_author / by_report / by_review',
  review_status  VARCHAR(16)     NOT NULL DEFAULT 'clean' COMMENT '机审：clean/pending（细化设计 7 节）',
  published_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '排序列',
  created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_post_time (status, published_at DESC),
  KEY idx_post_mood (status, mood, published_at DESC),
  KEY idx_post_author (author_id),
  CONSTRAINT fk_post_author FOREIGN KEY (author_id) REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='树洞帖';

-- ---------------------------------------------------------------------
-- 8. floor 楼层（楼中楼仅一层：有父楼层则 depth=2——CHECK 强制）
-- ---------------------------------------------------------------------
CREATE TABLE floor (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  post_id         BIGINT UNSIGNED NOT NULL,
  parent_floor_id BIGINT UNSIGNED NULL COMMENT '对楼层再回应',
  depth           TINYINT         NOT NULL DEFAULT 1 COMMENT '1 或 2',
  author_id       BIGINT UNSIGNED NOT NULL,
  content         VARCHAR(500)    NOT NULL,
  is_anonymous    TINYINT(1)      NOT NULL DEFAULT 1,
  status          VARCHAR(16)     NOT NULL DEFAULT 'published' COMMENT 'published/removed',
  review_status   VARCHAR(16)     NOT NULL DEFAULT 'clean' COMMENT 'clean/pending',
  created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_floor_post (post_id, created_at),
  KEY idx_floor_parent (parent_floor_id),
  CONSTRAINT chk_floor_pair CHECK (
    (parent_floor_id IS NULL AND depth = 1) OR
    (parent_floor_id IS NOT NULL AND depth = 2)
  ),
  CONSTRAINT fk_floor_post   FOREIGN KEY (post_id)         REFERENCES post (id),
  CONSTRAINT fk_floor_parent FOREIGN KEY (parent_floor_id) REFERENCES floor (id),
  CONSTRAINT fk_floor_author FOREIGN KEY (author_id)       REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='楼层';

-- ---------------------------------------------------------------------
-- 9. reaction 回应（toggle 由唯一约束实现；无负向类型——红线）
--    type/target 合法组合在应用层校验：post→hug,me_too；floor→hug,helpful
-- ---------------------------------------------------------------------
CREATE TABLE reaction (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  member_id   BIGINT UNSIGNED NOT NULL,
  target_type VARCHAR(8)      NOT NULL COMMENT 'post/floor',
  target_id   BIGINT UNSIGNED NOT NULL,
  type        VARCHAR(8)      NOT NULL COMMENT 'hug/me_too/helpful',
  created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_reaction (member_id, target_type, target_id, type),
  KEY idx_reaction_target (target_type, target_id),
  CONSTRAINT fk_reaction_member FOREIGN KEY (member_id) REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='回应';

-- ---------------------------------------------------------------------
-- 10. activity 活动
--     CHECK 约束报名数不超容量（管理员调小容量前需先核对，见注释）
-- ---------------------------------------------------------------------
CREATE TABLE activity (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  title          VARCHAR(40)     NOT NULL,
  type           VARCHAR(8)      NOT NULL COMMENT 'outdoor/pbl/drama/culture/salon',
  start_at       DATETIME(3)     NOT NULL,
  end_at         DATETIME(3)     NOT NULL,
  gather_place   VARCHAR(50)     NOT NULL DEFAULT '' COMMENT '集合地点',
  gather_time    VARCHAR(20)     NOT NULL DEFAULT '' COMMENT '集合时间（展示用）',
  age_range      VARCHAR(20)     NOT NULL DEFAULT '' COMMENT '适合年龄（展示用，如 7–12 岁）',
  price_cents    INT             NOT NULL DEFAULT 0 COMMENT '0=免费；第一阶段不接支付',
  capacity       INT             NOT NULL DEFAULT 0,
  enrolled_count INT             NOT NULL DEFAULT 0 COMMENT '由报名事务维护，禁止手工修改',
  status         VARCHAR(16)     NOT NULL DEFAULT 'draft' COMMENT 'draft/open/closed/finished',
  intro          TEXT            NULL,
  points         JSON            NULL COMMENT '会做什么：要点数组',
  bring          TEXT            NULL COMMENT '带什么',
  cover_art      VARCHAR(8)      NOT NULL DEFAULT 'leaf' COMMENT '封面插画主题 leaf/sun/grape/coral/sky',
  readonly_at    DATETIME(3)     NULL COMMENT '档案只读时间 = end_at + 30 天（CST 计算），巡检任务写入',
  created_by     BIGINT UNSIGNED NOT NULL,
  created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_activity_list (status, start_at),
  KEY idx_activity_type (type),
  CONSTRAINT chk_activity_quota CHECK (enrolled_count >= 0 AND enrolled_count <= capacity),
  CONSTRAINT fk_activity_creator FOREIGN KEY (created_by) REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='活动';

-- ---------------------------------------------------------------------
-- 11. resource_folder 资源文件夹（成员资料绑定活动——CHECK 强制）
-- ---------------------------------------------------------------------
CREATE TABLE resource_folder (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  title       VARCHAR(30)     NOT NULL,
  intro       VARCHAR(100)    NOT NULL DEFAULT '',
  visibility  VARCHAR(16)     NOT NULL DEFAULT 'member' COMMENT 'member=正式成员/activity=仅活动成员',
  activity_id BIGINT UNSIGNED NULL COMMENT 'visibility=activity 时必填',
  sort        INT             NOT NULL DEFAULT 0,
  created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_folder_sort (sort),
  CONSTRAINT chk_folder_activity CHECK (visibility <> 'activity' OR activity_id IS NOT NULL),
  CONSTRAINT fk_folder_activity FOREIGN KEY (activity_id) REFERENCES activity (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='资源文件夹';

-- ---------------------------------------------------------------------
-- 12. resource_file 资源文件（来源与更新时间是资源库可信度的一部分）
--     file_key 为本地盘相对路径（uploads/library/…），经 API 鉴权代理分发
-- ---------------------------------------------------------------------
CREATE TABLE resource_file (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  folder_id    BIGINT UNSIGNED NOT NULL,
  title        VARCHAR(100)    NOT NULL,
  file_type    VARCHAR(10)     NOT NULL COMMENT 'pdf/doc/xlsx/image',
  size_bytes   BIGINT          NOT NULL DEFAULT 0,
  read_minutes INT             NULL COMMENT '预计阅读时长（分钟）',
  source       VARCHAR(50)     NOT NULL DEFAULT '' COMMENT '管理员整理/家长共创',
  file_key     VARCHAR(255)    NOT NULL,
  status       VARCHAR(16)     NOT NULL DEFAULT 'active' COMMENT 'active/archived',
  created_by   BIGINT UNSIGNED NOT NULL,
  updated_at   DATETIME(3)     NULL,
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_file_folder (folder_id, status),
  CONSTRAINT fk_file_folder FOREIGN KEY (folder_id) REFERENCES resource_folder (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='资源文件';

-- ---------------------------------------------------------------------
-- 13. enrollment 报名（一条报名对应一个孩子；多孩子 = 多行）
--     报名防重：生成列 active_key 仅在 enrolled 时生成键（细化设计 2.2）；
--     MySQL 唯一索引允许多个 NULL，已取消记录不占约束
-- ---------------------------------------------------------------------
CREATE TABLE enrollment (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  activity_id  BIGINT UNSIGNED NOT NULL,
  member_id    BIGINT UNSIGNED NOT NULL,
  child_name   VARCHAR(20)     NOT NULL COMMENT '孩子称呼',
  child_age    TINYINT UNSIGNED NOT NULL,
  note         VARCHAR(200)    NULL COMMENT '想提前告诉老师的事',
  consent      JSON            NOT NULL COMMENT '[{type,version,agreed_at}]：safety 与 photo 两条',
  status       VARCHAR(16)     NOT NULL DEFAULT 'enrolled' COMMENT 'enrolled/cancelled',
  cancelled_at DATETIME(3)     NULL,
  active_key   VARCHAR(64)     GENERATED ALWAYS AS (
                   CASE WHEN status = 'enrolled'
                        THEN CONCAT(activity_id, ':', member_id, ':', child_name)
                        ELSE NULL END
               ) STORED,
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_enrollment_active (active_key),
  KEY idx_enroll_member (member_id, status),
  KEY idx_enroll_activity (activity_id, status),
  CONSTRAINT fk_enroll_activity FOREIGN KEY (activity_id) REFERENCES activity (id),
  CONSTRAINT fk_enroll_member   FOREIGN KEY (member_id)   REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='报名';

-- ---------------------------------------------------------------------
-- 14. activity_comment 活动讨论区（只聊本场活动）
-- ---------------------------------------------------------------------
CREATE TABLE activity_comment (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  activity_id   BIGINT UNSIGNED NOT NULL,
  author_id     BIGINT UNSIGNED NOT NULL,
  reply_to_id   BIGINT UNSIGNED NULL COMMENT '回复某条讨论',
  content       VARCHAR(300)    NOT NULL,
  status        VARCHAR(16)     NOT NULL DEFAULT 'published' COMMENT 'published/removed',
  review_status VARCHAR(16)     NOT NULL DEFAULT 'clean' COMMENT 'clean/pending',
  created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_comment_activity (activity_id, created_at),
  CONSTRAINT fk_comment_activity FOREIGN KEY (activity_id) REFERENCES activity (id),
  CONSTRAINT fk_comment_author   FOREIGN KEY (author_id)   REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='活动讨论区';

-- ---------------------------------------------------------------------
-- 15. archive_item 档案条目（照片/作品/手记统一建模；无评分字段——红线）
--     audience=all 集体内容；=specific 指定孩子，由 archive_audience 授权
--     file_key 为本地盘相对路径（uploads/archive/…），经 API 鉴权代理分发
-- ---------------------------------------------------------------------
CREATE TABLE archive_item (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  activity_id  BIGINT UNSIGNED NOT NULL,
  type         VARCHAR(8)      NOT NULL COMMENT 'photo/work/note',
  title        VARCHAR(60)     NOT NULL DEFAULT '' COMMENT '照片说明 / 作品名',
  file_key     VARCHAR(255)    NULL COMMENT '照片对象键',
  content      TEXT            NULL COMMENT '手记正文 / 作品说明',
  child_tag    VARCHAR(20)     NULL COMMENT '作品归属孩子称呼',
  audience     VARCHAR(8)      NOT NULL DEFAULT 'all' COMMENT 'all/specific',
  teacher_name VARCHAR(20)     NULL COMMENT '手记署名',
  teacher_role VARCHAR(20)     NULL COMMENT '手记角色（营地带队等）',
  created_by   BIGINT UNSIGNED NOT NULL COMMENT '上传的老师 / 管理员',
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_archive_activity (activity_id, type, created_at),
  CONSTRAINT fk_archive_activity FOREIGN KEY (activity_id) REFERENCES activity (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='档案条目';

-- ---------------------------------------------------------------------
-- 16. archive_audience 照片可见授权（授权到报名=孩子维度）
-- ---------------------------------------------------------------------
CREATE TABLE archive_audience (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  item_id       BIGINT UNSIGNED NOT NULL,
  enrollment_id BIGINT UNSIGNED NOT NULL,
  created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_audience (item_id, enrollment_id),
  KEY idx_audience_enroll (enrollment_id),
  CONSTRAINT fk_aud_item   FOREIGN KEY (item_id)       REFERENCES archive_item (id),
  CONSTRAINT fk_aud_enroll FOREIGN KEY (enrollment_id) REFERENCES enrollment (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='档案照片可见授权';

-- ---------------------------------------------------------------------
-- 17. message 私信（角色约束在应用层校验 + 每日巡检：至少一方为 teacher/admin——红线）
-- ---------------------------------------------------------------------
CREATE TABLE message (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  sender_id     BIGINT UNSIGNED NOT NULL,
  recipient_id  BIGINT UNSIGNED NOT NULL,
  content       VARCHAR(500)    NOT NULL,
  review_status VARCHAR(16)     NOT NULL DEFAULT 'clean' COMMENT 'clean/pending',
  read_at       DATETIME(3)     NULL,
  created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_msg_conv (sender_id, recipient_id, created_at),
  KEY idx_msg_inbox (recipient_id, read_at),
  CONSTRAINT fk_msg_sender    FOREIGN KEY (sender_id)    REFERENCES member (id),
  CONSTRAINT fk_msg_recipient FOREIGN KEY (recipient_id) REFERENCES member (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='私信';

-- ---------------------------------------------------------------------
-- 18. report 举报（目标多态不做外键；reporter_id 任何接口不回传——红线）
-- ---------------------------------------------------------------------
CREATE TABLE report (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  target_type VARCHAR(16)     NOT NULL COMMENT 'post/floor/message/resource_file',
  target_id   BIGINT UNSIGNED NOT NULL,
  reason      VARCHAR(8)      NOT NULL COMMENT 'privacy/attack/ads/other',
  detail      VARCHAR(200)    NULL,
  reporter_id BIGINT UNSIGNED NOT NULL COMMENT '内部保留，仅管理台可见',
  status      VARCHAR(16)     NOT NULL DEFAULT 'open' COMMENT 'open/handled/dismissed',
  handled_by  BIGINT UNSIGNED NULL,
  handled_at  DATETIME(3)     NULL,
  action      VARCHAR(50)     NULL COMMENT 'remove_content/no_action',
  note        VARCHAR(200)    NULL COMMENT '处置说明（no_action 必填）',
  created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_report_queue (status, created_at),
  KEY idx_report_target (target_type, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='举报';

-- ---------------------------------------------------------------------
-- 19. audit_log 审计（actor 可空、不设外键：审计独立于业务生命周期）
-- ---------------------------------------------------------------------
CREATE TABLE audit_log (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  actor_id    BIGINT UNSIGNED NULL COMMENT '操作者（系统任务为 NULL）',
  action      VARCHAR(50)     NOT NULL COMMENT 'phone.view/photo.view/admin.approve/export/...',
  target_type VARCHAR(16)     NULL,
  target_id   BIGINT UNSIGNED NULL,
  ip          VARCHAR(45)     NULL,
  user_agent  VARCHAR(255)    NULL,
  occurred_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_audit_actor (actor_id, occurred_at),
  KEY idx_audit_action (action, occurred_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='审计日志';
