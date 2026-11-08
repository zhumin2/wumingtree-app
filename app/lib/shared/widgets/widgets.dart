import 'package:flutter/material.dart';

import '../../core/theme/app_theme.dart';

/// 设计系统组件（M1-10）：从原型移植的厚底按钮 / 胶囊 / 贴纸 / 锁定卡 / 心情胶囊。
/// 全部走 WmtColors token，禁止页面内手搓一次性弹窗与写死色值（架构设计 4.5）。

/// 心情枚举与颜色映射（原型 moodChip()）。
class MoodDef {
  const MoodDef(this.code, this.label, this.color);

  final String code;
  final String label;
  final Color color;
}

const moodDefs = [
  MoodDef('share', '想分享', WmtColors.yellowDeep),
  MoodDef('tired', '有点累', WmtColors.skyDeep),
  MoodDef('ask', '求经验', WmtColors.greenDeep),
  MoodDef('talk', '只是想说说', WmtColors.coralDeep),
];

MoodDef moodOf(String code) =>
    moodDefs.firstWhere((m) => m.code == code, orElse: () => moodDefs.first);

/// 胶囊标签：颜色跟随模块语义；单卡不超过 4 个。
class WmtPill extends StatelessWidget {
  const WmtPill({super.key, required this.label, this.color = WmtColors.greenDeep, this.outline = false});

  final String label;
  final Color color;
  final bool outline;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: outline ? Colors.transparent : color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(12),
        border: outline ? Border.all(color: color) : null,
      ),
      child: Text(label, style: TextStyle(fontSize: 11, color: color, fontWeight: FontWeight.w600)),
    );
  }
}

/// 厚底按钮：实底 + 深色厚底投影，按下下沉 2px（原型 btn）。
/// 一屏只保留一个主操作；珊瑚色仅用于负向/终止操作。
class ThickButton extends StatefulWidget {
  const ThickButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.color = WmtColors.green,
    this.deep = WmtColors.greenDeep,
    this.expand = true,
    this.loading = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final Color color;
  final Color deep;
  final bool expand;
  final bool loading;

  @override
  State<ThickButton> createState() => _ThickButtonState();
}

class _ThickButtonState extends State<ThickButton> {
  bool _pressed = false;

  @override
  Widget build(BuildContext context) {
    final child = widget.loading
        ? const SizedBox(
            width: 20,
            height: 20,
            child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
        : Text(widget.label,
            style: const TextStyle(fontWeight: FontWeight.bold, color: Colors.white));
    final btn = GestureDetector(
      onTapDown: (_) => setState(() => _pressed = true),
      onTapUp: (_) => setState(() => _pressed = false),
      onTapCancel: () => setState(() => _pressed = false),
      onTap: widget.onPressed,
      child: Container(
        height: 52,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: widget.color,
          borderRadius: BorderRadius.circular(18),
          boxShadow: [
            BoxShadow(
              color: widget.deep,
              offset: Offset(0, _pressed ? 0 : 3),
            ),
          ],
        ),
        child: child,
      ),
    );
    return widget.expand ? SizedBox(width: double.infinity, child: btn) : btn;
  }
}

/// 贴纸：轻微旋转 -2°，仅用于日期、置顶等短文本（产品文档 9.3）。
class Sticker extends StatelessWidget {
  const Sticker({super.key, required this.text, this.color = WmtColors.yellow});

  final String text;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Transform.rotate(
      angle: -2 * 3.14159 / 180,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.18),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(text,
            style: TextStyle(
                fontSize: 11, color: WmtColors.ink, fontWeight: FontWeight.w600)),
      ),
    );
  }
}

/// 锁定卡：锁图标 + 说明 + 引导按钮（成员资料 / 档案未解锁）。
class LockedCard extends StatelessWidget {
  const LockedCard({super.key, required this.title, required this.hint, this.actionLabel, this.onAction});

  final String title;
  final String hint;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: WmtColors.grapeSoft,
        borderRadius: BorderRadius.circular(24),
      ),
      child: Column(
        children: [
          const Icon(Icons.lock_outline, size: 32, color: WmtColors.grapeDeep),
          const SizedBox(height: 12),
          Text(title,
              style: const TextStyle(fontWeight: FontWeight.bold, color: WmtColors.ink)),
          const SizedBox(height: 6),
          Text(hint,
              textAlign: TextAlign.center,
              style: const TextStyle(fontSize: 12, color: WmtColors.grapeDeep, height: 1.6)),
          if (actionLabel != null) ...[
            const SizedBox(height: 14),
            TextButton(onPressed: onAction, child: Text(actionLabel!)),
          ],
        ],
      ),
    );
  }
}

/// 心情胶囊（树洞卡片 / 发帖选择）；label 可覆盖（「全部」等非心情项）。
class MoodChip extends StatelessWidget {
  const MoodChip({super.key, required this.code, this.label, this.selected = false, this.onTap});

  final String code;
  final String? label;
  final bool selected;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final def = moodOf(code);
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: selected ? def.color.withValues(alpha: 0.15) : WmtColors.card,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: selected ? def.color : WmtColors.divider),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 6,
              height: 6,
              decoration: BoxDecoration(color: def.color, shape: BoxShape.circle),
            ),
            const SizedBox(width: 6),
            Text(label ?? def.label,
                style: TextStyle(
                    fontSize: 12,
                    color: def.color,
                    fontWeight: selected ? FontWeight.bold : FontWeight.normal)),
          ],
        ),
      ),
    );
  }
}

/// 失败重试态（列表 / 详情统一弱网组件）。
class ErrorRetry extends StatelessWidget {
  const ErrorRetry({super.key, required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(message, style: const TextStyle(color: WmtColors.skyDeep)),
          const SizedBox(height: 12),
          TextButton(onPressed: onRetry, child: const Text('重试')),
        ],
      ),
    );
  }
}
