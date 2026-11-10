import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'core/network/api_base.dart';
import 'core/network/api_client.dart';
import 'core/router/app_router.dart';
import 'core/router/url_strategy.dart';
import 'core/theme/app_theme.dart';
import 'features/auth/auth_controller.dart';
import 'features/auth/token_store.dart';
import 'features/hollow/hollow_controller.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  configureUrlStrategy();
  final auth = AuthController(
    store: TokenStore(),
    api: ApiClient(baseUrl: defaultBaseUrl()),
  );
  // 启动恢复令牌（M1-9）：成功 → member；失败 → guest。
  unawaited(auth.tryRestore());
  runApp(
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: auth),
        ChangeNotifierProvider(
            create: (_) => HollowController(auth.api)..refresh()),
      ],
      child: buildWmtApp(
        WmtAppDependencies(theme: buildWmtTheme(), router: buildRouter(auth)),
      ),
    ),
  );
}
