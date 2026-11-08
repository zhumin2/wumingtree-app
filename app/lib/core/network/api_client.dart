import 'package:dio/dio.dart';

/// 业务异常：信封 code != 0 时抛出，页面 catch 后展示 [message]。
class AppError implements Exception {
  AppError(this.code, this.message);

  final int code;
  final String message;

  @override
  String toString() => 'AppError($code, $message)';
}

/// 统一信封结构：{"code":0,"message":"ok","data":...,"request_id":"..."}
class Envelope {
  Envelope({required this.code, required this.message, this.data, this.requestId});

  factory Envelope.fromJson(Map<String, dynamic> j) => Envelope(
        code: (j['code'] as num?)?.toInt() ?? -1,
        message: (j['message'] as String?) ?? '',
        data: j['data'],
        requestId: (j['request_id'] as String?) ?? '',
      );

  final int code;
  final String message;
  final dynamic data;
  final String requestId;
}

/// 令牌供给钩子（AuthController 注入）：网络层不感知存储实现。
class TokenHooks {
  TokenHooks({this.current, this.refresh, this.onRevoked});

  /// 当前 access token（注入 Authorization）。
  final String? Function()? current;

  /// 收到 2001 时静默续期，返回新 access；null = 无法续期。
  final Future<String?> Function()? refresh;

  /// 收到 2002（会话被吊销）时的回调（清会话回登录）。
  final void Function()? onRevoked;
}

/// dio 封装：统一 get/post/patch/delete；`_unwrap` 解信封，code != 0 抛 [AppError]。
/// 2001 → 调用 refresh 钩子静默续期并重放一次；2002 → 回调吊销并抛出（架构设计 4.4）。
/// API 地址用 --dart-define=API_BASE_URL 编译期注入。
class ApiClient {
  ApiClient({required String baseUrl, TokenHooks? hooks})
      : _dio = Dio(
          BaseOptions(
            baseUrl: baseUrl,
            connectTimeout: const Duration(seconds: 10),
            receiveTimeout: const Duration(seconds: 15),
            headers: {'Accept': 'application/json'},
          ),
        ),
        hooks = hooks ?? TokenHooks();

  static const fallbackBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://127.0.0.1:8080/api/v1',
  );

  final Dio _dio;
  TokenHooks hooks;

  void setToken(String? token) {
    final t = (token == null || token.isEmpty) ? null : 'Bearer $token';
    _dio.options.headers['Authorization'] = t;
  }

  Future<T> get<T>(String path, {Map<String, dynamic>? query, T Function(dynamic)? parse}) =>
      _send((dio) => dio.get<Map<String, dynamic>>(path, queryParameters: query), parse);

  Future<T> post<T>(String path, {Object? body, T Function(dynamic)? parse}) =>
      _send((dio) => dio.post<Map<String, dynamic>>(path, data: body), parse);

  Future<T> patch<T>(String path, {Object? body, T Function(dynamic)? parse}) =>
      _send((dio) => dio.patch<Map<String, dynamic>>(path, data: body), parse);

  Future<T> delete<T>(String path, {T Function(dynamic)? parse}) =>
      _send((dio) => dio.delete<Map<String, dynamic>>(path), parse);

  Future<T> _send<T>(
    Future<Response<Map<String, dynamic>>> Function(Dio dio) fn,
    T Function(dynamic)? parse,
  ) async {
    final env = await _once(fn);
    // 2001：静默续期后重放一次
    if (env.code == 2001) {
      final renewed = await hooks.refresh?.call();
      if (renewed != null && renewed.isNotEmpty) {
        final retried = await _once(fn);
        return _finish(retried, parse);
      }
    }
    return _finish(env, parse);
  }

  Future<Envelope> _once(Future<Response<Map<String, dynamic>>> Function(Dio dio) fn) async {
    final Response<Map<String, dynamic>> res;
    try {
      res = await fn(_dio);
    } on DioException {
      // 网络层错误（信封之外），统一转为弱提示
      throw AppError(-1, '网络不太顺畅，稍后再试试');
    }
    return Envelope.fromJson(res.data ?? <String, dynamic>{});
  }

  Future<T> _finish<T>(Envelope env, T Function(dynamic)? parse) async {
    switch (env.code) {
      case 0:
        return parse != null ? parse(env.data) : env.data as T;
      case 2002:
        hooks.onRevoked?.call();
        throw AppError(env.code, env.message);
      default:
        throw AppError(env.code, env.message);
    }
  }
}
