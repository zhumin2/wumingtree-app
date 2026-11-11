import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wumingtree_app/features/agreement/agreement_page.dart';
import 'package:wumingtree_app/features/gate/gate_page.dart';

void main() {
  group('AgreementPage', () {
    testWidgets('渲染隐私政策分节正文', (tester) async {
      await tester.pumpWidget(const MaterialApp(home: AgreementPage(doc: 'privacy')));
      expect(find.text('隐私政策'), findsOneWidget);
      expect(find.text('三、我们如何存储与保护'), findsOneWidget);
      expect(find.textContaining('仅管理员可见，家长之间互不可见'), findsOneWidget);
    });

    testWidgets('未知 doc 回退用户协议（深链兜底）', (tester) async {
      await tester.pumpWidget(
          const MaterialApp(home: AgreementPage(doc: 'not-exist')));
      expect(find.text('用户协议'), findsOneWidget);
    });
  });

  group('GatePage 协议勾选', () {
    Future<void> fillValidForm(WidgetTester tester) async {
      await tester.enterText(
          find.widgetWithText(TextFormField, '邀请码（WMT-XXXX-XXXX）'), 'WMT-TEST-0001');
      await tester.enterText(
          find.widgetWithText(TextFormField, '怎么称呼你'), '测试家长');
      await tester.enterText(
          find.widgetWithText(TextFormField, '手机号（仅管理员可见，家长之间互不可见）'),
          '13800138000');
      await tester.pump();
    }

    testWidgets('未勾选提交被阻断并提示', (tester) async {
      await tester.pumpWidget(const MaterialApp(home: GatePage()));
      await fillValidForm(tester);
      await tester.tap(find.text('提交申请'));
      await tester.pump();
      expect(find.text('请先阅读并勾选同意协议'), findsOneWidget);
    });

    testWidgets('勾选后 Checkbox 状态切换', (tester) async {
      await tester.pumpWidget(const MaterialApp(home: GatePage()));
      expect(tester.widget<Checkbox>(find.byType(Checkbox)).value, isFalse);
      await tester.tap(find.byType(Checkbox));
      await tester.pump();
      expect(tester.widget<Checkbox>(find.byType(Checkbox)).value, isTrue);
    });
  });
}
