import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/network/api_client.dart';
import '../../core/theme/app_theme.dart';
import '../../shared/models/models.dart';
import '../../shared/widgets/widgets.dart';
import 'hollow_list_page.dart' show PostCard;
import '../auth/auth_controller.dart';
import 'hollow_controller.dart';

/// 我的树洞：自己发过的帖子（含匿名帖显示为自己的记录——红线）；可删帖。
class MyPostsPage extends StatefulWidget {
  const MyPostsPage({super.key});

  @override
  State<MyPostsPage> createState() => _MyPostsPageState();
}

class _MyPostsPageState extends State<MyPostsPage> {
  List<HollowPost> _posts = [];
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final auth = context.read<AuthController>();
      _posts = await HollowController(auth.api).myPosts();
    } on AppError catch (e) {
      _error = e.message;
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _delete(HollowPost post) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('删除这条帖子？'),
        content: const Text('回应会一并隐藏，删除后不可恢复。'),
        actions: [
          TextButton(onPressed: () => Navigator.of(ctx).pop(false), child: const Text('取消')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: WmtColors.coralDeep),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      final auth = context.read<AuthController>();
      await HollowController(auth.api).deletePost(post.id);
      _load();
    } on AppError catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(e.message)));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('我的树洞')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? ErrorRetry(message: _error!, onRetry: _load)
              : _posts.isEmpty
                  ? const Center(
                      child: Text('还没有发过帖子',
                          style: TextStyle(color: WmtColors.skyDeep)))
                  : ListView(
                      padding: const EdgeInsets.all(16),
                      children: [
                        for (final p in _posts)
                          PostCard(post: p, onLongPress: () => _delete(p)),
                      ],
                    ),
    );
  }
}
