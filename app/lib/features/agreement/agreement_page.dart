import 'package:flutter/material.dart';

import '../../core/theme/app_theme.dart';
import 'agreement_content.dart';

/// 协议静态页：用户协议 / 隐私政策 / 社区规范（上线清单 9.1 第 6 项）。
/// 渲染沿用 GatePendingPage 静态页模式（AppBar + ListView + WmtColors），零新依赖；
/// Web 形态同代码库可用（不引 webview）。
class AgreementPage extends StatelessWidget {
  const AgreementPage({super.key, required this.doc});

  /// 文档标识：user / privacy / rules（见 [kAgreements]）。
  final String doc;

  @override
  Widget build(BuildContext context) {
    final d = agreementDoc(doc);
    return Scaffold(
      appBar: AppBar(title: Text(d.title)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          for (final s in d.sections) ...[
            Text(s.heading,
                style: const TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.bold,
                    color: WmtColors.ink)),
            const SizedBox(height: 8),
            for (final p in s.paragraphs) ...[
              Text(p,
                  style: const TextStyle(
                      fontSize: 14, height: 1.7, color: WmtColors.ink)),
              const SizedBox(height: 6),
            ],
            const SizedBox(height: 14),
          ],
          const SizedBox(height: 8),
          const Center(
            child: Text('只对邀请开放的家长陪伴社群\n不排名 · 不打卡 · 不比孩子',
                textAlign: TextAlign.center,
                style:
                    TextStyle(fontSize: 12, color: WmtColors.skyDeep, height: 1.8)),
          ),
        ],
      ),
    );
  }
}
