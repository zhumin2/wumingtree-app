import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'app.dart';
import 'core/router/app_router.dart';
import 'core/theme/app_theme.dart';
import 'features/auth/auth_controller.dart';
import 'features/auth/token_store.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final auth = AuthController(store: TokenStore());
  // 启动恢复令牌（M1-9）：成功 → member；失败 → guest。
  unawaited(auth.tryRestore());
  runApp(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: auth),
      ],
      child: buildWmtApp(
        WmtAppDependencies(theme: buildWmtTheme(), router: buildRouter(auth)),
      ),
    ),
  );
}
