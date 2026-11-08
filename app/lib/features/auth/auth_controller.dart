import 'package:flutter/foundation.dart';

import '../../core/network/api_client.dart';
import '../../shared/models/models.dart';
import 'token_store.dart';

/// 会话状态，驱动路由 refreshListenable 重定向（架构设计 4.3）。
enum AuthStatus { unknown, guest, pending, member }

/// 全局会话 Controller（MultiProvider 挂载，架构设计 4.2）。
/// 负责令牌存取、静默续期、登录/登出状态机；页面只 watch 不碰 dio。
class AuthController extends ChangeNotifier {
  AuthController({ApiClient? api, TokenStore? store})
      : api = api ?? ApiClient(baseUrl: ApiClient.fallbackBaseUrl),
        store = store ?? TokenStore() {
    wireHooks();
  }

  final ApiClient api;
  final TokenStore store;

  AuthStatus _status = AuthStatus.unknown;
  Me? _me;
  String? _pendingReason;
  bool _loading = false;

  AuthStatus get status => _status;
  Me? get me => _me;
  String? get pendingReason => _pendingReason;
  bool get loading => _loading;

  void wireHooks() {
    api.hooks = TokenHooks(
      current: () => _accessToken,
      refresh: () => _refresh(),
      onRevoked: () => forceGuest(reason: '登录已过期，请重新登录'),
    );
  }

  String? _accessToken;
  String? _refreshToken;

  void _setLoading(bool v) {
    _loading = v;
    notifyListeners();
  }

  /// 启动恢复：有令牌则校验 /me；2001 走静默续期；失败一律 guest。
  Future<void> tryRestore() async {
    final access = await store.access();
    final refresh = await store.refresh();
    if (access == null || access.isEmpty) {
      _status = AuthStatus.guest;
      notifyListeners();
      return;
    }
    _accessToken = access;
    _refreshToken = refresh;
    api.setToken(access);
    try {
      _me = await api.get<Me>('/me', parse: Me.fromJson);
      _status = AuthStatus.member;
    } on AppError catch (e) {
      if (e.code == 2001 && await _refresh() != null) {
        try {
          _me = await api.get<Me>('/me', parse: Me.fromJson);
          _status = AuthStatus.member;
        } on AppError {
          await forceGuest();
          return;
        }
      } else {
        await forceGuest();
        return;
      }
    }
    notifyListeners();
  }

  /// 提交加入申请（公开端点）。成功后调用方进入审核状态页。
  Future<void> submitApplication({
    required String inviteCode,
    required String nickname,
    required String phone,
  }) async {
    _setLoading(true);
    try {
      await api.post<dynamic>('/applications', body: {
        'inviteCode': inviteCode,
        'nickname': nickname,
        'phone': phone,
      });
    } finally {
      _setLoading(false);
    }
  }

  /// 发送验证码（60s 频控由服务端强制；4001 时页面提示稍后再试）。
  Future<void> sendCode(String phone) async {
    await api.post<dynamic>('/auth/sms-codes', body: {'phone': phone});
  }

  /// 更新本人资料（设置页开关与昵称；服务端校验，失败原值回滚由 watch 重渲染完成）。
  Future<void> patchMe({
    String? nickname,
    bool? defaultAnon,
    bool? dnd,
    bool? notifyNotice,
    bool? notifyHollow,
  }) async {
    final body = <String, dynamic>{
      if (nickname != null) 'nickname': nickname,
      if (defaultAnon != null) 'defaultAnon': defaultAnon,
      if (dnd != null) 'dnd': dnd,
      if (notifyNotice != null) 'notifyNotice': notifyNotice,
      if (notifyHollow != null) 'notifyHollow': notifyHollow,
    };
    _me = await api.patch<Me>('/me', body: body, parse: Me.fromJson);
    notifyListeners();
  }

  /// 登录：按 outcome 分流（member / pending / rejected / no_record）。
  Future<LoginResult> login(String phone, String code) async {
    _setLoading(true);
    try {
      final res = await api.post<LoginResult>(
        '/auth/login',
        body: {'phone': phone, 'code': code},
        parse: LoginResult.fromJson,
      );
      if (res.isMember && res.tokens != null) {
        _accessToken = res.tokens!.accessToken;
        _refreshToken = res.tokens!.refreshToken;
        await store.save(access: _accessToken!, refresh: _refreshToken ?? '');
        api.setToken(_accessToken);
        _me = res.me ?? await api.get<Me>('/me', parse: Me.fromJson);
        _status = AuthStatus.member;
        notifyListeners();
      } else if (res.isPending || res.isRejected) {
        _pendingReason = res.application?.rejectReason;
        _status = AuthStatus.pending;
        notifyListeners();
      }
      return res;
    } finally {
      _setLoading(false);
    }
  }

  /// 静默续期：用 refresh token 轮换，成功返回新 access。
  Future<String?> _refresh() async {
    final rt = _refreshToken ?? await store.refresh();
    if (rt == null || rt.isEmpty) return null;
    try {
      final tokens = await api.post<Tokens>(
        '/auth/refresh',
        body: {'refreshToken': rt},
        parse: Tokens.fromJson,
      );
      _accessToken = tokens.accessToken;
      _refreshToken = tokens.refreshToken;
      await store.save(access: _accessToken!, refresh: _refreshToken!);
      api.setToken(_accessToken);
      notifyListeners();
      return _accessToken;
    } on AppError {
      return null;
    }
  }

  /// 退出登录（M1 本地清除；服务端吊销随 M5 私信一并接入）。
  Future<void> logout() => forceGuest();

  /// 会话失效：清空并回到未登录。
  Future<void> forceGuest({String? reason}) async {
    _accessToken = null;
    _refreshToken = null;
    _me = null;
    _pendingReason = reason;
    api.setToken(null);
    await store.clear();
    _status = AuthStatus.guest;
    notifyListeners();
  }
}
