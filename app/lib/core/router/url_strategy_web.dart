// Web 目标：hash 策略（/static/web/#/home），由 Go API 同源托管时
// 无需 nginx try_files，也不受 --base-href 子路径影响。
import 'package:flutter_web_plugins/flutter_web_plugins.dart';

void configureUrlStrategy() {
  setUrlStrategy(const HashUrlStrategy());
}
