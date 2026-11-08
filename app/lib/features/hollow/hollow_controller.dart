import 'package:flutter/foundation.dart';

import '../../core/network/api_client.dart';
import '../../shared/models/models.dart';

/// 树洞列表状态（心情筛选 + 游标分页；时间倒序——红线）。
class HollowController extends ChangeNotifier {
  HollowController(this.api);

  final ApiClient api;

  List<HollowPost> items = [];
  String? nextCursor;
  String mood = ''; // 空 = 全部
  bool loading = false;
  String? error;

  Future<void> refresh({String? mood}) async {
    if (mood != null) this.mood = mood;
    loading = true;
    error = null;
    notifyListeners();
    try {
      final page = await api.get<PostPage>('/posts', query: {
        if (this.mood.isNotEmpty) 'mood': this.mood,
      }, parse: PostPage.fromJson);
      items = page.items;
      nextCursor = page.nextCursor;
    } on AppError catch (e) {
      error = e.message;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  Future<void> loadMore() async {
    if (nextCursor == null || loading) return;
    loading = true;
    notifyListeners();
    try {
      final page = await api.get<PostPage>('/posts', query: {
        if (mood.isNotEmpty) 'mood': mood,
        'cursor': nextCursor!,
      }, parse: PostPage.fromJson);
      items = [...items, ...page.items];
      nextCursor = page.nextCursor;
    } on AppError catch (e) {
      error = e.message;
    } finally {
      loading = false;
      notifyListeners();
    }
  }

  /// 帖子详情 + 首屏楼层（crisis=true 时页面立即叠加求助卡）。
  Future<PostDetail> getPost(int postId) {
    return api.get<PostDetail>('/posts/$postId', parse: PostDetail.fromJson);
  }

  /// 我的树洞（服务端按作者过滤；匿名帖显示为自己的记录——红线）。
  Future<List<HollowPost>> myPosts({String? cursor}) {
    return api.get<List<HollowPost>>('/me/posts', query: {
      if (cursor != null) 'cursor': cursor,
    }, parse: (data) {
      final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
      return ((j['items'] as List?) ?? []).map(HollowPost.fromJson).toList();
    });
  }

  /// 发帖（client_token 幂等；重试复用同一 token——架构设计 3.4）。
  Future<HollowPost> createPost({
    required String mood,
    required String title,
    String? body,
    required bool anonymous,
  }) {
    final token = newClientToken();
    return api.post<HollowPost>(
      '/posts',
      body: {
        'mood': mood,
        'title': title,
        if (body != null && body.isNotEmpty) 'body': body,
        'isAnonymous': anonymous,
      },
      headers: {'client_token': token},
      parse: HollowPost.fromJson,
    );
  }

  /// 发表楼层（parentFloorId 对楼层回应，楼中楼仅一层）。
  Future<HollowFloor> createFloor({
    required int postId,
    required String content,
    int? parentFloorId,
    required bool anonymous,
  }) {
    final token = newClientToken();
    return api.post<HollowFloor>(
      '/posts/$postId/floors',
      body: {
        'content': content,
        if (parentFloorId != null) 'parentFloorId': parentFloorId,
        'isAnonymous': anonymous,
      },
      headers: {'client_token': token},
      parse: HollowFloor.fromJson,
    );
  }

  /// 帖子级回应 toggle（hug 抱抱 / me_too 我也是）。
  Future<ReactionState> togglePostReaction(int postId, String type) {
    return api.post<ReactionState>('/posts/$postId/reactions',
        body: {'type': type}, parse: ReactionState.fromJson);
  }

  /// 楼层级回应 toggle（hug 抱一下 / helpful 有帮助）。
  Future<ReactionState> toggleFloorReaction(int floorId, String type) {
    return api.post<ReactionState>('/floors/$floorId/reactions',
        body: {'type': type}, parse: ReactionState.fromJson);
  }

  /// 删帖（回应一并隐藏，不可恢复）。
  Future<void> deletePost(int postId) => api.delete<dynamic>('/posts/$postId');

  /// 举报（四项原因；举报人身份服务端不回传——红线）。
  Future<void> report(int postId, String reason, String? detail) {
    return api.post<dynamic>('/reports', body: {
      'targetType': 'post',
      'targetId': postId,
      'reason': reason,
      if (detail != null && detail.isNotEmpty) 'detail': detail,
    });
  }
}
