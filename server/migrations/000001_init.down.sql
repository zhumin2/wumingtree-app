-- =====================================================================
-- 无名树 App · 初始迁移回滚（000001_init.down.sql）
-- 与 000001_init.up.sql 成对（golang-migrate 管理）。
-- 红线：生产库有真实数据后禁止执行 down（架构设计 3.7：一切脚本禁止 DROP/DELETE）。
-- 按外键依赖逆序删除。
-- =====================================================================
SET NAMES utf8mb4;

DROP TABLE IF EXISTS archive_audience;
DROP TABLE IF EXISTS archive_item;
DROP TABLE IF EXISTS activity_comment;
DROP TABLE IF EXISTS enrollment;
DROP TABLE IF EXISTS resource_file;
DROP TABLE IF EXISTS resource_folder;
DROP TABLE IF EXISTS activity;
DROP TABLE IF EXISTS reaction;
DROP TABLE IF EXISTS floor;
DROP TABLE IF EXISTS post;
DROP TABLE IF EXISTS notice_read;
DROP TABLE IF EXISTS notice;
DROP TABLE IF EXISTS application;
DROP TABLE IF EXISTS invite_code;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS message;
DROP TABLE IF EXISTS report;
DROP TABLE IF EXISTS member_secure;
DROP TABLE IF EXISTS member;
