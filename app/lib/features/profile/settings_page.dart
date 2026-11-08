import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/network/api_client.dart';
import '../../core/theme/app_theme.dart';
import '../auth/auth_controller.dart';

/// 版本号：--dart-define=APP_VERSION 注入；与仓库口径统一（技术设计 12 章）。
const _appVersion = String.fromEnvironment('APP_VERSION', defaultValue: '0.1.0');

/// 设置页（M1-12）：昵称、默认匿名、免打扰、提醒开关、退出。
/// 服务端为准：开关即时 PATCH /me，失败回滚。
class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key});

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthController>();
    final me = auth.me;
    return Scaffold(
      appBar: AppBar(title: const Text('设置')),
      body: me == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                Card(
                  color: WmtColors.card,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(18),
                      side: const BorderSide(color: WmtColors.divider)),
                  child: ListTile(
                    title: const Text('昵称'),
                    subtitle: Text(me.nickname),
                    trailing: const Icon(Icons.edit, size: 18),
                    onTap: () => _editNickname(context, me.nickname),
                  ),
                ),
                _SwitchTile(
                    title: '默认匿名发帖',
                    subtitle: '发帖时显示「一位家长」',
                    value: me.defaultAnon,
                    onChanged: (v) => _patch(context, defaultAnon: v)),
                _SwitchTile(
                    title: '免打扰',
                    subtitle: '22:00–08:00 不提醒',
                    value: me.dnd,
                    onChanged: (v) => _patch(context, dnd: v)),
                _SwitchTile(
                    title: '公告板提醒',
                    subtitle: '只有管理员发新公告时提醒',
                    value: me.notifyNotice,
                    onChanged: (v) => _patch(context, notifyNotice: v)),
                _SwitchTile(
                    title: '树洞回应提醒',
                    subtitle: '有人回应你的帖子时提醒',
                    value: me.notifyHollow,
                    onChanged: (v) => _patch(context, notifyHollow: v)),
                const SizedBox(height: 8),
                Card(
                  color: WmtColors.card,
                  shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(18),
                      side: const BorderSide(color: WmtColors.divider)),
                  child: ListTile(
                    title: const Text('退出登录',
                        style: TextStyle(color: WmtColors.coralDeep)),
                    onTap: () => _confirmLogout(context),
                  ),
                ),
                const SizedBox(height: 16),
                const Center(
                  child: Text('无名树 v$_appVersion',
                      style: TextStyle(fontSize: 12, color: WmtColors.skyDeep)),
                ),
              ],
            ),
    );
  }

  Future<void> _patch(BuildContext context,
      {bool? defaultAnon, bool? dnd, bool? notifyNotice, bool? notifyHollow}) async {
    final auth = context.read<AuthController>();
    try {
      await auth.patchMe(
          defaultAnon: defaultAnon,
          dnd: dnd,
          notifyNotice: notifyNotice,
          notifyHollow: notifyHollow);
    } on AppError catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.message)));
      }
    }
  }

  Future<void> _editNickname(BuildContext context, String current) async {
    final controller = TextEditingController(text: current);
    final auth = context.read<AuthController>();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('修改昵称'),
        content: TextField(
          controller: controller,
          maxLength: 8,
          autofocus: true,
          decoration: const InputDecoration(
              hintText: '其他人看到的名字，手机号不会显示', counterText: ''),
        ),
        actions: [
          TextButton(onPressed: () => ctx.pop(false), child: const Text('取消')),
          FilledButton(
              style: FilledButton.styleFrom(backgroundColor: WmtColors.green),
              onPressed: () => ctx.pop(true),
              child: const Text('保存')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await auth.patchMe(nickname: controller.text.trim());
    } on AppError catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.message)));
      }
    }
  }

  Future<void> _confirmLogout(BuildContext context) async {
    final auth = context.read<AuthController>();
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('退出登录？'),
        content: const Text('下次进入需要重新用短信验证码登录。'),
        actions: [
          TextButton(onPressed: () => ctx.pop(false), child: const Text('再待一会')),
          FilledButton(
              style: FilledButton.styleFrom(backgroundColor: WmtColors.coralDeep),
              onPressed: () => ctx.pop(true),
              child: const Text('退出')),
        ],
      ),
    );
    if (ok != true) return;
    await auth.logout();
    if (context.mounted) context.go('/gate');
  }
}

class _SwitchTile extends StatelessWidget {
  const _SwitchTile({
    required this.title,
    required this.subtitle,
    required this.value,
    required this.onChanged,
  });

  final String title;
  final String subtitle;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: WmtColors.card,
      margin: const EdgeInsets.only(top: 8),
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(18),
          side: const BorderSide(color: WmtColors.divider)),
      child: SwitchListTile(
        title: Text(title, style: const TextStyle(color: WmtColors.ink)),
        subtitle: Text(subtitle, style: const TextStyle(fontSize: 12, color: WmtColors.skyDeep)),
        value: value,
        onChanged: onChanged,
      ),
    );
  }
}
