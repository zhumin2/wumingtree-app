import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/network/api_client.dart';
import '../../core/theme/app_theme.dart';
import '../../shared/widgets/widgets.dart';
import 'hollow_controller.dart';

/// 发帖页：心情选择 + 一句标题（必填）+ 正文（可选）+ 匿名开关（默认开）。
/// 提交前提示不留可识别信息；client_token 幂等（架构设计 3.4）。
class ComposePage extends StatefulWidget {
  const ComposePage({super.key});

  @override
  State<ComposePage> createState() => _ComposePageState();
}

class _ComposePageState extends State<ComposePage> {
  String _mood = 'tired';
  bool _anon = true;
  bool _sending = false;
  final _title = TextEditingController();
  final _body = TextEditingController();

  @override
  void dispose() {
    _title.dispose();
    _body.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final title = _title.text.trim();
    if (title.isEmpty) {
      _toast('写一句标题吧');
      return;
    }
    setState(() => _sending = true);
    try {
      final post = await context.read<HollowController>().createPost(
            mood: _mood,
            title: title,
            body: _body.text.trim(),
            anonymous: _anon,
          );
      if (!mounted) return;
      await context.read<HollowController>().refresh();
      if (mounted) context.go('/hollow/posts/${post.id}');
    } on AppError catch (e) {
      _toast(e.message);
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('说说此刻')),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          const Text('现在的心情', style: TextStyle(fontSize: 13, color: WmtColors.skyDeep)),
          const SizedBox(height: 10),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final m in moodDefs)
                MoodChip(code: m.code, selected: _mood == m.code, onTap: () => setState(() => _mood = m.code)),
            ],
          ),
          const SizedBox(height: 20),
          TextField(
            controller: _title,
            maxLength: 50,
            decoration: const InputDecoration(
                labelText: '写一句标题（必填）', hintText: '不用解释全部背景，先说第一句', counterText: ''),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _body,
            maxLength: 2000,
            maxLines: 6,
            decoration: const InputDecoration(
                labelText: '想多说几句（可选）',
                hintText: '这里不评判、不建议。也请不要留下孩子学校、住址等可识别信息',
                counterText: ''),
          ),
          const SizedBox(height: 8),
          SwitchListTile(
            title: const Text('匿名发布',
                style: TextStyle(color: WmtColors.ink, fontWeight: FontWeight.w600)),
            subtitle: Text(
                _anon ? '显示为「一位家长」' : '实名发布 · 显示你的昵称',
                style: const TextStyle(fontSize: 12, color: WmtColors.skyDeep)),
            value: _anon,
            onChanged: (v) => setState(() => _anon = v),
          ),
          const SizedBox(height: 16),
          ThickButton(
            label: '发布',
            color: WmtColors.yellow,
            deep: WmtColors.yellowDeep,
            loading: _sending,
            onPressed: _submit,
          ),
          const SizedBox(height: 12),
          const Text(
            '发布后其他家长可以回应。如果情况涉及自伤、暴力或医疗紧急，请先联系专业机构，而不是等待回应。',
            style: TextStyle(fontSize: 12, color: WmtColors.skyDeep, height: 1.6),
          ),
        ],
      ),
    );
  }
}
