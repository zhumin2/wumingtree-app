# app · 无名树 App 客户端（Flutter）

一套代码出 Android / iOS。架构与约定见《无名树App技术架构设计-v1.0》第四章；
UI 验收锚点是仓库根目录的高保真原型 `index.html`。

## 首次初始化

```bash
cd app
flutter create --platforms=android,ios --org app.wumingtree .
flutter pub get
flutter analyze && flutter test
```

`flutter create` 会生成 `android/`、`ios/` 平台目录（生成后随仓库提交）。

## API 地址注入

```bash
# Android 联调（本机服务）
flutter build apk --dart-define=API_BASE_URL=http://10.0.2.2:8080/api/v1
# iOS 模拟器（默认值即 127.0.0.1:8080，无需注入）
flutter run
```

版本号注入：`--dart-define=APP_VERSION=x.y.z`（技术设计 12 章口径统一）。
