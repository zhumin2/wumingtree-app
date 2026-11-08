import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/app_theme.dart';

/// 首页骨架（M0 占位；M2 起接入真实四入口数据）。
/// 风险提示条仅首页常驻，不随内容滚动（产品红线，技术设计 2.4）。
class HomePage extends StatelessWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('无名树'),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_outline),
            tooltip: '我的',
            onPressed: () => context.push('/me'),
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              padding: const EdgeInsets.all(16),
              children: const [
                _EntryCard(color: WmtColors.yellowDeep, title: '公告板', subtitle: '置顶：11 月集合时间'),
                _EntryCard(color: WmtColors.coralDeep, title: '树洞互助', subtitle: '今天有人说话了'),
                _EntryCard(color: WmtColors.skyDeep, title: '经验资源库', subtitle: '5 个主题 · 长期留存'),
                _EntryCard(color: WmtColors.greenDeep, title: '活动专区', subtitle: '5 场正在报名与排期'),
              ],
            ),
          ),
          const RiskStrip(),
        ],
      ),
    );
  }
}

class _EntryCard extends StatelessWidget {
  const _EntryCard({required this.color, required this.title, required this.subtitle});

  final Color color;
  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Card(
      color: WmtColors.card,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24), side: const BorderSide(color: WmtColors.divider)),
      child: ListTile(
        title: Text(title, style: const TextStyle(fontWeight: FontWeight.bold, color: WmtColors.ink)),
        subtitle: Text(subtitle, style: const TextStyle(color: WmtColors.skyDeep)),
        trailing: Icon(Icons.arrow_forward_ios, size: 16, color: color),
      ),
    );
  }
}

/// 常驻风险提示条：文案为产品红线，集中在客户端常量（技术设计 2.4）。
class RiskStrip extends StatelessWidget {
  const RiskStrip({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      color: WmtColors.yellowSoft,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: const Text(
        '同伴互助不能替代专业医疗或心理支持，紧急情况请及时就医',
        style: TextStyle(fontSize: 12, color: WmtColors.yellowDeep),
        textAlign: TextAlign.center,
      ),
    );
  }
}
