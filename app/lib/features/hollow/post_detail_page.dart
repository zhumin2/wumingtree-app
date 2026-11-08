import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/network/api_client.dart';
import '../../core/theme/app_theme.dart';
import '../../shared/models/models.dart';
import '../../shared/widgets/widgets.dart';
import 'hollow_controller.dart';

/// 帖子与楼层：楼层从上到下；抱一下/我也是 toggle；举报抽屉；危机卡（红线）。
class PostDetailPage extends StatefulWidget {
  const PostDetailPage({super.key, required this.postId});

  final int postId;

  @override
  State<PostDetailPage> createState() => _PostDetailPageState();
}

class _PostDetailPageState extends State<PostDetailPage> {
  late Future<PostDetail> _future;
  final _floorInput = TextEditingController();
  bool _anonFloor = true;
  bool _sending = false;

  @override
  void initState() {
    super.initState();
    _future = context.read<HollowController>().getPost(widget.postId);
  }

  void _reload() {
    setState(() {
      _future = context.read<HollowController>().getPost(widget.postId);
    });
  }

  @override
  void dispose() {
    _floorInput.dispose();
    super.dispose();
  }

  Future<void> _togglePost(String type) async {
    try {
      await context.read<HollowController>().togglePostReaction(widget.postId, type);
      _reload();
    } on AppError catch (e) {
      _toast(e.message);
    }
  }

  Future<void> _toggleFloor(int floorId) async {
    try {
      await context.read<HollowController>().toggleFloorReaction(floorId, 'hug');
      _reload();
    } on AppError catch (e) {
      _toast(e.message);
    }
  }

  Future<void> _sendFloor() async {
    final content = _floorInput.text.trim();
    if (content.isEmpty) return;
    setState(() => _sending = true);
    try {
      await context.read<HollowController>().createFloor(
            postId: widget.postId,
            content: content,
            anonymous: _anonFloor,
          );
      _floorInput.clear();
      _reload();
    } on AppError catch (e) {
      _toast(e.message);
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  Future<void> _report(int postId) async {
    final reason = await showModalBottomSheet<String>(
      context: context,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Padding(
              padding: EdgeInsets.all(16),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text('举报这条内容',
                    style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
              ),
            ),
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 16),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text('管理员会看到，发帖人不会知道是谁举报的。',
                    style: TextStyle(fontSize: 12, color: WmtColors.skyDeep)),
              ),
            ),
            for (final r in const [
              ('privacy', '泄露了孩子或家庭的隐私'),
              ('attack', '带有评判、指责或攻击'),
              ('ads', '广告、引流或与社群无关'),
              ('other', '其他'),
            ])
              ListTile(title: Text(r.$2), onTap: () => ctx.pop(r.$1)),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
    if (reason == null) return;
    try {
      await context.read<HollowController>().report(postId, reason, null);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('已提交举报，管理员会处理')));
      }
    } on AppError catch (e) {
      if (mounted) _toast(e.message);
    }
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('帖子'),
        actions: [
          IconButton(
            icon: const Icon(Icons.flag_outlined),
            tooltip: '举报',
            onPressed: () => _report(widget.postId),
          ),
        ],
      ),
      bottomSheet: _floorInputBar(),
      body: FutureBuilder<PostDetail>(
        future: _future,
        builder: (context, snap) {
          if (snap.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snap.hasError) {
            final e = snap.error;
            final msg = e is AppError ? e.message : '加载失败';
            return ErrorRetry(message: msg, onRetry: _reload);
          }
          final d = snap.data!;
          return ListView(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 96),
            children: [
              if (d.crisis) ...[
                const CrisisCard(),
                const SizedBox(height: 12),
              ],
              _PostHeader(post: d.post, onToggle: _togglePost),
              const SizedBox(height: 16),
              const Divider(color: WmtColors.divider),
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 8),
                child: Text('回应',
                    style: TextStyle(fontWeight: FontWeight.bold, color: WmtColors.ink)),
              ),
              if (d.floors.isEmpty)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 24),
                  child: Center(
                      child: Text('还没有回应。有人会看到你的。',
                          style: TextStyle(color: WmtColors.skyDeep))),
                ),
              for (final f in d.floors)
                _FloorTile(
                    floor: f, isReply: f.depth == 2, onHug: () => _toggleFloor(f.id)),
            ],
          );
        },
      ),
    );
  }

  Widget _floorInputBar() {
    return SafeArea(
      child: Container(
        color: WmtColors.card,
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
        child: Row(
          children: [
            Expanded(
              child: TextField(
                controller: _floorInput,
                maxLength: 500,
                decoration: InputDecoration(
                    hintText: '先接住，不评判、不建议', counterText: '', isDense: true),
              ),
            ),
            const SizedBox(width: 8),
            IconButton(
              icon: Icon(
                _anonFloor ? Icons.visibility_off_outlined : Icons.visibility_outlined,
                color: WmtColors.skyDeep,
              ),
              tooltip: _anonFloor ? '匿名回应' : '实名回应',
              onPressed: () => setState(() => _anonFloor = !_anonFloor),
            ),
            IconButton(
              icon: const Icon(Icons.send, color: WmtColors.coralDeep),
              onPressed: _sending ? null : _sendFloor,
            ),
          ],
        ),
      ),
    );
  }
}

/// 危机求助卡：危机词命中的帖子立即叠加（红线：先联系专业机构，不等待回应）。
/// 文案由产品统一维护；热线号码需产品确认后补充。
class CrisisCard extends StatelessWidget {
  const CrisisCard({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: WmtColors.coralSoft,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: WmtColors.coral),
      ),
      child: const Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            Icon(Icons.health_and_safety_outlined, size: 18, color: WmtColors.coralDeep),
            SizedBox(width: 8),
            Text('你现在安全吗？',
                style: TextStyle(fontWeight: FontWeight.bold, color: WmtColors.coralDeep)),
          ]),
          SizedBox(height: 8),
          Text(
            '如果情况涉及自伤、暴力或医疗紧急，请先联系专业机构或拨打当地心理援助热线，不要等待社群回应。这里不是替代专业帮助的地方。',
            style: TextStyle(fontSize: 12, color: WmtColors.ink, height: 1.7),
          ),
        ],
      ),
    );
  }
}

class _PostHeader extends StatelessWidget {
  const _PostHeader({required this.post, required this.onToggle});

  final HollowPost post;
  final void Function(String type) onToggle;

  @override
  Widget build(BuildContext context) {
    final hugged = post.myReactions.contains('hug');
    final meToo = post.myReactions.contains('me_too');
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(children: [MoodChip(code: post.mood)]),
        const SizedBox(height: 10),
        Text(post.title,
            style: const TextStyle(
                fontSize: 18, fontWeight: FontWeight.bold, color: WmtColors.ink, height: 1.5)),
        if (post.body.isNotEmpty) ...[
          const SizedBox(height: 10),
          Text(post.body,
              style: const TextStyle(fontSize: 14, color: WmtColors.ink, height: 1.8)),
        ],
        const SizedBox(height: 14),
        Row(children: [
          Text(post.displayName,
              style: const TextStyle(fontSize: 12, color: WmtColors.skyDeep)),
          const Spacer(),
          TextButton.icon(
            onPressed: () => onToggle('hug'),
            icon: Icon(
                hugged ? Icons.favorite : Icons.favorite_outline,
                size: 16,
                color: WmtColors.coralDeep),
            label: Text('抱一下 ${post.hug}',
                style: const TextStyle(fontSize: 12, color: WmtColors.coralDeep)),
          ),
          TextButton.icon(
            onPressed: () => onToggle('me_too'),
            icon: Icon(meToo ? Icons.thumb_up : Icons.thumb_up_outlined,
                size: 16, color: WmtColors.greenDeep),
            label: Text('我也是 ${post.meToo}',
                style: const TextStyle(fontSize: 12, color: WmtColors.greenDeep)),
          ),
        ]),
      ],
    );
  }
}

class _FloorTile extends StatelessWidget {
  const _FloorTile({required this.floor, required this.isReply, required this.onHug});

  final HollowFloor floor;
  final bool isReply;
  final VoidCallback onHug;

  @override
  Widget build(BuildContext context) {
    final hugged = floor.hug > 0; // 简化：有无 hug 计数（toggle 后重载可见）
    return Padding(
      padding: EdgeInsets.only(left: isReply ? 24 : 0, bottom: 12),
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: isReply ? WmtColors.greenSoft : WmtColors.card,
          borderRadius: BorderRadius.circular(18),
          border: Border.all(color: WmtColors.divider),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Text(floor.displayName,
                  style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13, color: WmtColors.ink)),
              const Spacer(),
              TextButton.icon(
                onPressed: onHug,
                icon: Icon(hugged ? Icons.favorite : Icons.favorite_outline,
                    size: 14, color: WmtColors.coralDeep),
                label: Text('${floor.hug}',
                    style: const TextStyle(fontSize: 12, color: WmtColors.coralDeep)),
              ),
            ]),
            const SizedBox(height: 6),
            Text(floor.content,
                style: const TextStyle(fontSize: 14, color: WmtColors.ink, height: 1.7)),
          ],
        ),
      ),
    );
  }
}
