// 应用级装配说明：根组件与依赖在 `core/router/app_router.dart` 的
// buildWmtApp / WmtAppDependencies 中定义，`main.dart` 负责 MultiProvider 装配。
//
// 本文件保留导出，保持 import 路径稳定（架构设计 4.1）。
export 'core/router/app_router.dart' show buildWmtApp, WmtAppDependencies;
