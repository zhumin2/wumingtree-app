import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/auth_controller.dart';
import '../../features/auth/login_page.dart';
import '../../features/gate/gate_page.dart';
import '../../features/home/home_page.dart';
import '../../features/profile/profile_page.dart';
import '../../features/profile/settings_page.dart';

/// 路由表集中于此（架构设计 4.3）。铁律：二级页面一律 context.push()，
/// go() 只用于登录成功进首页、退出登录回 gate 等平级重置场景。
GoRouter buildRouter(AuthController auth) {
  return GoRouter(
    refreshListenable: auth,
    initialLocation: '/home',
    redirect: (context, state) {
      final path = state.matchedLocation;
      switch (auth.status) {
        case AuthStatus.unknown:
          return null; // 启动期不做跳转
        case AuthStatus.member:
          // 成员不得停留在 gate / login
          if (path.startsWith('/gate') || path == '/login') return '/home';
          return null;
        case AuthStatus.pending:
          if (path == '/gate/pending') return null;
          if (path == '/login' || path == '/gate') return null; // 可回到登录/邀请页
          return '/gate/pending';
        case AuthStatus.guest:
          // guest 只能到 gate / login（含其子路径）
          if (path.startsWith('/gate') || path == '/login') return null;
          return '/gate';
      }
    },
    routes: [
      GoRoute(
        path: '/gate',
        builder: (context, state) => const GatePage(),
        routes: [
          GoRoute(path: 'pending', builder: (context, state) => const GatePendingPage()),
        ],
      ),
      GoRoute(path: '/login', builder: (context, state) => const LoginPage()),
      GoRoute(path: '/home', builder: (_, __) => const HomePage()),
      GoRoute(path: '/me', builder: (_, __) => const ProfilePage()),
      GoRoute(path: '/me/settings', builder: (_, __) => const SettingsPage()),
    ],
  );
}

Widget buildWmtApp(WmtAppDependencies deps) {
  return MaterialApp.router(
    title: '无名树',
    debugShowCheckedModeBanner: false,
    theme: deps.theme,
    routerConfig: deps.router,
  );
}

/// 依赖注入容器：main.dart 装配（便于测试）。
class WmtAppDependencies {
  const WmtAppDependencies({required this.theme, required this.router});

  final ThemeData theme;
  final GoRouter router;
}
