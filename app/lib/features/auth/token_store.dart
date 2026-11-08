import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// 令牌安全存储（flutter_secure_storage；架构设计 4.2）。
class TokenStore {
  static const _access = 'wmt.access';
  static const _refresh = 'wmt.refresh';

  final _s = const FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );

  Future<String?> access() => _s.read(key: _access);
  Future<String?> refresh() => _s.read(key: _refresh);

  Future<void> save({required String access, required String refresh}) async {
    await _s.write(key: _access, value: access);
    await _s.write(key: _refresh, value: refresh);
  }

  Future<void> clear() async {
    await _s.delete(key: _access);
    await _s.delete(key: _refresh);
  }
}
