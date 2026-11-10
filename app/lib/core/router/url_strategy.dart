// URL 策略：Web 用 hash 路由（免疫 base-href 子路径深链问题，参考架构 4.4）。
// 条件导入：dart:html 仅存在于 Web 编译目标。

export 'url_strategy_stub.dart'
    if (dart.library.html) 'url_strategy_web.dart';
