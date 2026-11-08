/// 与服务端 VO 一一对应的模型层（架构设计 4.1 shared/models）。
/// 字段命名 snake_case（信封）→ camelCase（Dart）的映射集中在此，禁止散落。
library;

/// 登录结果（openapi LoginResult）。
class LoginResult {
  LoginResult({required this.outcome, this.tokens, this.me, this.application});

  factory LoginResult.fromJson(dynamic data) {
    final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
    return LoginResult(
      outcome: (j['outcome'] as String?) ?? 'no_record',
      tokens: j['tokens'] == null ? null : Tokens.fromJson(j['tokens']),
      me: j['member'] == null ? null : Me.fromJson(j['member']),
      application: j['application'] == null ? null : ApplicationInfo.fromJson(j['application']),
    );
  }

  final String outcome;
  final Tokens? tokens;
  final Me? me;
  final ApplicationInfo? application;

  bool get isMember => outcome == 'member';
  bool get isPending => outcome == 'application_pending';
  bool get isRejected => outcome == 'application_rejected';
  bool get isNoRecord => outcome == 'no_record';
}

/// 令牌对。
class Tokens {
  Tokens({required this.accessToken, required this.refreshToken});

  factory Tokens.fromJson(dynamic data) {
    final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
    return Tokens(
      accessToken: (j['accessToken'] as String?) ?? '',
      refreshToken: (j['refreshToken'] as String?) ?? '',
    );
  }

  final String accessToken;
  final String refreshToken;
}

/// 我的资料。服务端契约保证无明文手机号，maskedPhone 仅展示（红线）。
class Me {
  Me({
    required this.id,
    required this.nickname,
    required this.role,
    required this.defaultAnon,
    required this.dnd,
    required this.notifyNotice,
    required this.notifyHollow,
    required this.joinedAt,
    required this.maskedPhone,
  });

  factory Me.fromJson(dynamic data) {
    final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
    return Me(
      id: (j['id'] as num).toInt(),
      nickname: (j['nickname'] as String?) ?? '',
      role: (j['role'] as String?) ?? 'member',
      defaultAnon: (j['defaultAnon'] as bool?) ?? true,
      dnd: (j['dnd'] as bool?) ?? true,
      notifyNotice: (j['notifyNotice'] as bool?) ?? true,
      notifyHollow: (j['notifyHollow'] as bool?) ?? true,
      joinedAt: DateTime.tryParse((j['joinedAt'] as String?) ?? '') ?? DateTime.now(),
      maskedPhone: (j['maskedPhone'] as String?) ?? '',
    );
  }

  final int id;
  final String nickname;
  final String role;
  final bool defaultAnon;
  final bool dnd;
  final bool notifyNotice;
  final bool notifyHollow;
  final DateTime joinedAt;
  final String maskedPhone;

  /// 已加入天数（我的页身份卡）。
  int get joinedDays => DateTime.now().difference(joinedAt).inDays + 1;
}

/// 申请进度（登录 outcome 附带 / 提交申请返回）。
class ApplicationInfo {
  ApplicationInfo({
    required this.id,
    required this.status,
    required this.submittedAt,
    this.rejectReason,
    this.reviewedAt,
  });

  factory ApplicationInfo.fromJson(dynamic data) {
    final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
    return ApplicationInfo(
      id: (j['id'] as num).toInt(),
      status: (j['status'] as String?) ?? 'submitted',
      submittedAt: DateTime.tryParse((j['submittedAt'] as String?) ?? '') ?? DateTime.now(),
      rejectReason: j['rejectReason'] as String?,
      reviewedAt: j['reviewedAt'] == null ? null : DateTime.tryParse(j['reviewedAt'] as String),
    );
  }

  final int id;
  final String status;
  final DateTime submittedAt;
  final String? rejectReason;
  final DateTime? reviewedAt;
}

/// 树洞帖（匿名帖无 authorId 字段——红线）。
class HollowPost {
  HollowPost({
    required this.id,
    required this.mood,
    required this.title,
    required this.body,
    required this.isAnonymous,
    required this.displayName,
    required this.replies,
    required this.hug,
    required this.meToo,
    required this.myReactions,
    required this.publishedAt,
  });

  factory HollowPost.fromJson(dynamic data) {
    final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
    return HollowPost(
      id: (j['id'] as num).toInt(),
      mood: (j['mood'] as String?) ?? 'share',
      title: (j['title'] as String?) ?? '',
      body: (j['body'] as String?) ?? '',
      isAnonymous: (j['isAnonymous'] as bool?) ?? true,
      displayName: (j['displayName'] as String?) ?? '一位家长',
      replies: ((j['replies'] as num?) ?? 0).toInt(),
      hug: ((j['hug'] as num?) ?? 0).toInt(),
      meToo: ((j['meToo'] as num?) ?? 0).toInt(),
      myReactions: ((j['myReactions'] as List?) ?? [])
          .map((e) => e as String)
          .toList(),
      publishedAt: DateTime.tryParse((j['publishedAt'] as String?) ?? '') ?? DateTime.now(),
    );
  }

  final int id;
  final String mood;
  final String title;
  final String body;
  final bool isAnonymous;
  final String displayName;
  final int replies;
  final int hug;
  final int meToo;
  final List<String> myReactions;
  final DateTime publishedAt;
}

/// 楼层。
class HollowFloor {
  HollowFloor({
    required this.id,
    required this.parentFloorId,
    required this.depth,
    required this.displayName,
    required this.content,
    required this.hug,
    required this.helpful,
    required this.createdAt,
  });

  factory HollowFloor.fromJson(dynamic data) {
    final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
    return HollowFloor(
      id: (j['id'] as num).toInt(),
      parentFloorId: j['parentFloorId'] == null ? null : (j['parentFloorId'] as num).toInt(),
      depth: ((j['depth'] as num?) ?? 1).toInt(),
      displayName: (j['displayName'] as String?) ?? '一位家长',
      content: (j['content'] as String?) ?? '',
      hug: ((j['hug'] as num?) ?? 0).toInt(),
      helpful: ((j['helpful'] as num?) ?? 0).toInt(),
      createdAt: DateTime.tryParse((j['createdAt'] as String?) ?? '') ?? DateTime.now(),
    );
  }

  final int id;
  final int? parentFloorId;
  final int depth;
  final String displayName;
  final String content;
  final int hug;
  final int helpful;
  final DateTime createdAt;
}

/// 帖子详情（crisis=true 时立即叠加求助卡——红线）。
class PostDetail {
  PostDetail({required this.post, required this.floors, required this.crisis});

  factory PostDetail.fromJson(dynamic data) {
    final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
    return PostDetail(
      post: HollowPost.fromJson(j['post']),
      floors: ((j['floors'] as List?) ?? []).map(HollowFloor.fromJson).toList(),
      crisis: (j['crisis'] as bool?) ?? false,
    );
  }

  final HollowPost post;
  final List<HollowFloor> floors;
  final bool crisis;
}

/// 回应 toggle 结果。
class ReactionState {
  ReactionState({required this.active, required this.count});

  factory ReactionState.fromJson(dynamic data) {
    final j = (data ?? <String, dynamic>{}) as Map<String, dynamic>;
    return ReactionState(
      active: (j['active'] as bool?) ?? false,
      count: ((j['count'] as num?) ?? 0).toInt(),
    );
  }

  final bool active;
  final int count;
}

/// 客户端幂等令牌（写操作携带，重试复用同一 token——架构设计 3.4）。
String newClientToken() {
  final r = DateTime.now().microsecondsSinceEpoch;
  final buf = StringBuffer();
  var seed = r;
  for (var i = 0; i < 16; i++) {
    seed = (seed * 6364136223846793005 + 1442695040888963407) & 0x7FFFFFFFFFFFFFFF;
    buf.write((seed >> 33 & 0xFF).toRadixString(16).padLeft(2, '0'));
  }
  return buf.toString();
}
