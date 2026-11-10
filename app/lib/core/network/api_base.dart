import 'package:flutter/foundation.dart';

/// API 基地址（架构设计 4.3/参考架构 4.4）：
/// - Web 构建用相对路径 /api/v1（同源托管，否则打到访客自己机器）
/// - Android APK 用公网地址、本地联调用 127.0.0.1
/// - 同一代码库多产物多地址，全靠 --dart-define 注入
String defaultBaseUrl() {
  if (kIsWeb) return '/api/v1';
  return const String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://127.0.0.1:8080/api/v1',
  );
}
