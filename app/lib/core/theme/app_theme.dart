import 'package:flutter/material.dart';

/// 色板 token —— 与《无名树App技术设计文档-v1.0》3.3 表一一对应。
/// 铁律：业务代码禁止写死色值，一律引用这里（架构设计 4.5）。
abstract final class WmtColors {
  static const surface = Color(0xFFFFFBF0); // 奶油白 · 页面底
  static const card = Color(0xFFFFFFFF); // 卡片白
  static const ink = Color(0xFF1C3428); // 墨绿黑 · 主文字

  static const green = Color(0xFF12A566); // 嫩叶绿 · 主操作/活动/成功
  static const greenDeep = Color(0xFF0B7A4A);
  static const greenSoft = Color(0xFFE2F8EB);

  static const yellow = Color(0xFFFFC63D); // 太阳黄 · 公告/提醒/名额
  static const yellowDeep = Color(0xFFE5A31A);
  static const yellowSoft = Color(0xFFFFF3CC);

  static const coral = Color(0xFFFF7A5C); // 珊瑚橙 · 树洞/情绪/已满
  static const coralDeep = Color(0xFFE85B3E);
  static const coralSoft = Color(0xFFFFE9E2);

  static const sky = Color(0xFF3AA9F0); // 天空蓝 · 资源库/隐私说明
  static const skyDeep = Color(0xFF1E85C8);
  static const skySoft = Color(0xFFE1F4FF);

  static const grape = Color(0xFF8A7BE6); // 葡萄紫 · 戏剧/作品/成员资料
  static const grapeDeep = Color(0xFF6E5FD0);
  static const grapeSoft = Color(0xFFECE8FD);

  static const divider = Color(0xFFF2E8D3); // 分隔线（暖调）
}

/// 全局主题：明亮、圆润、有手绘感（产品文档 9.1）。
/// 圆角四档：12（小标签）/ 18（输入框）/ 24（内容卡）/ 30（主卡片）。
ThemeData buildWmtTheme() {
  final base = ThemeData(
    useMaterial3: true,
    colorScheme: ColorScheme.fromSeed(
      seedColor: WmtColors.green,
      surface: WmtColors.surface,
    ),
    scaffoldBackgroundColor: WmtColors.surface,
  );
  return base.copyWith(
    appBarTheme: const AppBarTheme(
      backgroundColor: WmtColors.surface,
      foregroundColor: WmtColors.ink,
      elevation: 0,
      centerTitle: true,
    ),
    dividerColor: WmtColors.divider,
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: WmtColors.card,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(18),
        borderSide: const BorderSide(color: WmtColors.divider),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(18),
        borderSide: const BorderSide(color: WmtColors.divider),
      ),
    ),
  );
}
