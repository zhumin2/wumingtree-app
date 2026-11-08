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
