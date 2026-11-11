import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/theme/app_theme.dart';
import '../auth/auth_controller.dart';

/// 我的页（M1-11 骨架）：身份卡 + 菜单；不做社交主页（产品文档 6.9）。
class ProfilePage extends StatelessWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthController>();
    final me = auth.me;
    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // 身份卡
          Card(
            color: WmtColors.card,
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(24),
                side: const BorderSide(color: WmtColors.divider)),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  CircleAvatar(
                    radius: 28,
                    backgroundColor: WmtColors.greenSoft,
                    child: Text(
                      (me?.nickname.isNotEmpty ?? false)
                          ? me!.nickname.characters.first
                          : '家',
                      style: const TextStyle(
                          color: WmtColors.greenDeep,
                          fontWeight: FontWeight.bold,
                          fontSize: 20),
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(me?.nickname ?? '…',
                            style: const TextStyle(
                                fontWeight: FontWeight.bold,
                                fontSize: 18,
                                color: WmtColors.ink)),
                        const SizedBox(height: 4),
                        Text(
                          '默认匿名 · 手机号不公开 · 已加入 ${me?.joinedDays ?? 0} 天',
                          style: const TextStyle(
                              fontSize: 12, color: WmtColors.skyDeep),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          _MenuTile(
              icon: Icons.event_note,
              title: '我的报名',
              onTap: () => _todo(context)), // M4 接线
          _MenuTile(
              icon: Icons.forum,
              title: '我的树洞',
              onTap: () => context.push('/me/posts')),
          _MenuTile(
              icon: Icons.palette,
              title: '孩子的作品',
              onTap: () => _todo(context)), // M4 接线
          _MenuTile(
              icon: Icons.mail,
              title: '私信',
              onTap: () => _todo(context)), // M5 接线
          _MenuTile(
              icon: Icons.settings,
              title: '设置',
              onTap: () => context.push('/me/settings')),
          // 协议入口（上线清单 9.1 第 6 项）：静态正文页，游客与成员均可访问。
          _MenuTile(
              icon: Icons.description_outlined,
              title: '用户协议',
              onTap: () => context.push('/agreement/user')),
          _MenuTile(
              icon: Icons.privacy_tip_outlined,
              title: '隐私政策',
              onTap: () => context.push('/agreement/privacy')),
          _MenuTile(
              icon: Icons.rule_outlined,
              title: '社区规范',
              onTap: () => context.push('/agreement/rules')),
          const SizedBox(height: 24),
          const Center(
            child: Text(
              '只对邀请开放的家长陪伴社群\n不排名 · 不打卡 · 不比孩子',
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 12, color: WmtColors.skyDeep, height: 1.8),
            ),
          ),
        ],
      ),
    );
  }

  void _todo(BuildContext context) {
    ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('这个模块在接下来的版本开放')));
  }
}

class _MenuTile extends StatelessWidget {
  const _MenuTile({required this.icon, required this.title, required this.onTap});

  final IconData icon;
  final String title;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: WmtColors.card,
      margin: const EdgeInsets.only(bottom: 8),
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(18),
          side: const BorderSide(color: WmtColors.divider)),
      child: ListTile(
        leading: Icon(icon, color: WmtColors.greenDeep, size: 22),
        title: Text(title, style: const TextStyle(color: WmtColors.ink)),
        trailing: const Icon(Icons.arrow_forward_ios, size: 14, color: WmtColors.divider),
        onTap: onTap,
      ),
    );
  }
}
