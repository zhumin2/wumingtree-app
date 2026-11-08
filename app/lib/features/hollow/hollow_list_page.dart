import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/theme/app_theme.dart';
import '../../shared/models/models.dart';
import '../../shared/widgets/widgets.dart';
import 'hollow_controller.dart';

/// 树洞列表：心情筛选（四种）+ 时间倒序 + 空态（产品文档 6.4）。
class HollowListPage extends StatefulWidget {
  const HollowListPage({super.key});

  @override
  State<HollowListPage> createState() => _HollowListPageState();
}

class _HollowListPageState extends State<HollowListPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<HollowController>().refresh();
    });
  }

  @override
  Widget build(BuildContext context) {
    final c = context.watch<HollowController>();
    return Scaffold(
      appBar: AppBar(
        title: const Text('树洞互助'),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_outline),
            tooltip: '我的',
            onPressed: () => context.push('/me'),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/hollow/compose'),
        backgroundColor: WmtColors.coral,
        foregroundColor: Colors.white,
        icon: const Icon(Icons.edit_note),
        label: const Text('说说此刻'),
      ),
      body: Column(
        children: [
          // 心情筛选：全部 + 四种心情；不使用横向滚动（技术设计 3.4）
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                MoodChip(code: '', label: '全部', selected: c.mood.isEmpty, onTap: () => c.refresh(mood: '')),
                for (final m in moodDefs)
                  MoodChip(code: m.code, selected: c.mood == m.code, onTap: () => c.refresh(mood: m.code)),
              ],
            ),
          ),
          Expanded(child: _buildBody(context, c)),
        ],
      ),
    );
  }

  Widget _buildBody(BuildContext context, HollowController c) {
    if (c.loading && c.items.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (c.error != null && c.items.isEmpty) {
      return ErrorRetry(message: c.error!, onRetry: () => c.refresh());
    }
    if (c.items.isEmpty) {
      return const EmptyHollow();
    }
    return RefreshIndicator(
      color: WmtColors.coralDeep,
      onRefresh: () => c.refresh(),
      child: ListView.builder(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 88),
        itemCount: c.items.length + (c.nextCursor != null ? 1 : 0),
        itemBuilder: (context, i) {
          if (i >= c.items.length) {
            WidgetsBinding.instance.addPostFrameCallback((_) => c.loadMore());
            return const Padding(
              padding: EdgeInsets.all(16),
              child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
            );
          }
          return PostCard(post: c.items[i]);
        },
      ),
    );
  }
}

/// 帖子卡片：心情胶囊 + 标题 + 摘要 + 匿名标识 + 回应数（无热度排序）。
class PostCard extends StatelessWidget {
  const PostCard({super.key, required this.post, this.onLongPress});

  final HollowPost post;
  final VoidCallback? onLongPress;

  @override
  Widget build(BuildContext context) {
    final def = moodOf(post.mood);
    return Card(
      color: WmtColors.card,
      margin: const EdgeInsets.only(bottom: 10),
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(24),
          side: const BorderSide(color: WmtColors.divider)),
      child: InkWell(
        borderRadius: BorderRadius.circular(24),
        onLongPress: onLongPress,
        onTap: () => context.push('/hollow/posts/${post.id}'),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  MoodChip(code: post.mood),
                  const Spacer(),
                  Text(post.displayName,
                      style: const TextStyle(fontSize: 12, color: WmtColors.skyDeep)),
                ],
              ),
              const SizedBox(height: 10),
              Text(post.title,
                  style: const TextStyle(
                      fontWeight: FontWeight.bold, fontSize: 15, color: WmtColors.ink)),
              if (post.body.isNotEmpty) ...[
                const SizedBox(height: 6),
                Text(post.body,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 13, color: WmtColors.skyDeep, height: 1.6)),
              ],
              const SizedBox(height: 10),
              Row(
                children: [
                  const Icon(Icons.chat_bubble_outline, size: 14, color: WmtColors.divider),
                  const SizedBox(width: 4),
                  Text('${post.replies}',
                      style: const TextStyle(fontSize: 12, color: WmtColors.skyDeep)),
                  const SizedBox(width: 14),
                  const Icon(Icons.favorite_outline, size: 14, color: WmtColors.coral),
                  const SizedBox(width: 4),
                  Text('${post.hug}',
                      style: const TextStyle(fontSize: 12, color: WmtColors.skyDeep)),
                  const Spacer(),
                  Text(
                      '${post.publishedAt.month}-${post.publishedAt.day.toString().padLeft(2, '0')}',
                      style: const TextStyle(fontSize: 11, color: WmtColors.divider)),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 树洞空态：先接住，不给压力（产品文档 6.4）。
class EmptyHollow extends StatelessWidget {
  const EmptyHollow({super.key});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.spa_outlined, size: 44, color: WmtColors.coral),
          const SizedBox(height: 12),
          const Text('还没有人说，第一句可以说给你听',
              style: TextStyle(color: WmtColors.skyDeep)),
        ],
      ),
    );
  }
}

