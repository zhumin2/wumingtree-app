import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wumingtree_app/core/network/api_client.dart';
import 'package:wumingtree_app/core/theme/app_theme.dart';

void main() {
  group('AppError', () {
    test('携带业务码与用户可读文案', () {
      final e = AppError(4201, '名额已满');
      expect(e.code, 4201);
      expect(e.message, '名额已满');
    });
  });

  group('WmtColors', () {
    test('色板 token 与设计文档一致', () {
      // 《技术设计文档》3.3：主色嫩叶绿、底奶油白、主文字墨绿黑。
      expect(WmtColors.surface, const Color(0xFFFFFBF0));
      expect(WmtColors.green, const Color(0xFF12A566));
      expect(WmtColors.ink, const Color(0xFF1C3428));
      expect(WmtColors.divider, const Color(0xFFF2E8D3));
    });
  });

  group('Envelope', () {
    test('解析统一信封', () {
      final env = Envelope.fromJson({
        'code': 0,
        'message': 'ok',
        'data': {'k': 'v'},
        'request_id': 'req-1',
      });
      expect(env.code, 0);
      expect(env.requestId, 'req-1');
    });
  });
}
