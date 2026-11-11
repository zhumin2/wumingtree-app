import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/network/api_client.dart';
import '../../core/theme/app_theme.dart';
import '../auth/auth_controller.dart';

/// 邀请进入页：品牌主张 + 申请表单（M1-8）。
/// 文案口径：不承诺改变人生；明确「不会推送营销信息」「手机号仅管理员可见」。
class GatePage extends StatefulWidget {
  const GatePage({super.key});

  @override
  State<GatePage> createState() => _GatePageState();
}

class _GatePageState extends State<GatePage> {
  final _formKey = GlobalKey<FormState>();
  final _invite = TextEditingController();
  final _nickname = TextEditingController();
  final _phone = TextEditingController();
  bool _submitting = false;
  bool _agreed = false;

  @override
  void dispose() {
    _invite.dispose();
    _nickname.dispose();
    _phone.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (!_agreed) {
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('请先阅读并勾选同意协议')));
      return;
    }
    setState(() => _submitting = true);
    final auth = context.read<AuthController>();
    try {
      await auth.submitApplication(
        inviteCode: _invite.text.trim(),
        nickname: _nickname.text.trim(),
        phone: _phone.text.trim(),
      );
      if (mounted) context.push('/gate/pending');
    } on AppError catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.message)));
      }
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Form(
          key: _formKey,
          child: ListView(
            padding: const EdgeInsets.all(24),
            children: [
              const SizedBox(height: 24),
              Text('在树下，慢慢长大',
                  style: Theme.of(context)
                      .textTheme
                      .headlineMedium
                      ?.copyWith(fontWeight: FontWeight.bold, color: WmtColors.ink)),
              const SizedBox(height: 8),
              const Text('一个只对邀请开放的家长陪伴社群，服务正在休整期的孩子与家庭。',
                  style: TextStyle(color: WmtColors.skyDeep)),
              const SizedBox(height: 32),
              TextFormField(
                controller: _invite,
                decoration: const InputDecoration(labelText: '邀请码（WMT-XXXX-XXXX）'),
                validator: (v) =>
                    (v == null || v.trim().length < 4) ? '请输入邀请码' : null,
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _nickname,
                maxLength: 8,
                decoration: const InputDecoration(
                    labelText: '怎么称呼你', counterText: ''),
                validator: (v) =>
                    (v == null || v.trim().isEmpty) ? '请填写称呼（最多 8 字）' : null,
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _phone,
                keyboardType: TextInputType.phone,
                decoration: const InputDecoration(
                    labelText: '手机号（仅管理员可见，家长之间互不可见）'),
                validator: (v) =>
                    RegExp(r'^1[3-9]\d{9}$').hasMatch((v ?? '').trim())
                        ? null
                        : '请填写正确的手机号',
              ),
              const SizedBox(height: 12),
              const Text(
                '提交后由管理员人工审核，通常 1–2 天。不会向你推送任何营销信息。',
                style: TextStyle(fontSize: 12, color: WmtColors.skyDeep),
              ),
              const SizedBox(height: 16),
              // 协议勾选（上线清单 9.1 第 6 项）：正文在 /agreement/:doc，guest 可访问。
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SizedBox(
                    width: 24,
                    height: 24,
                    child: Checkbox(
                      value: _agreed,
                      onChanged: (v) => setState(() => _agreed = v ?? false),
                      visualDensity: VisualDensity.compact,
                      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      activeColor: WmtColors.green,
                      side: const BorderSide(color: WmtColors.divider),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Wrap(
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        const Text('我已阅读并同意',
                            style: _consentStyle),
                        _agreementLink(context, '《用户协议》', '/agreement/user'),
                        const Text('、', style: _consentStyle),
                        _agreementLink(
                            context, '《隐私政策》', '/agreement/privacy'),
                        const Text('和', style: _consentStyle),
                        _agreementLink(
                            context, '《社区规范》', '/agreement/rules'),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 24),
              SizedBox(
                height: 52,
                child: FilledButton(
                  style: FilledButton.styleFrom(
                      backgroundColor: WmtColors.green,
                      foregroundColor: Colors.white,
                      shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(18))),
                  onPressed: _submitting ? null : _submit,
                  child: _submitting
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                      : const Text('提交申请'),
                ),
              ),
              const SizedBox(height: 12),
              TextButton(
                onPressed: () => context.push('/login'),
                child: const Text('已经是成员？用短信验证码登录',
                    style: TextStyle(color: WmtColors.greenDeep)),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 协议勾选行的正文样式（13 号、天蓝深、行高 1.5，与勾选框对齐）。
const _consentStyle = TextStyle(fontSize: 13, color: WmtColors.skyDeep, height: 1.5);

/// 协议链接片段：绿色深、可点，跳转 /agreement/:doc。
/// 用 GestureDetector + Wrap 拼接而非 TextSpan recognizer，避免手势识别器的生命周期管理。
Widget _agreementLink(BuildContext context, String label, String path) {
  return GestureDetector(
    behavior: HitTestBehavior.opaque,
    onTap: () => context.push(path),
    child: Text(label,
        style: const TextStyle(
            fontSize: 13, height: 1.5, color: WmtColors.greenDeep)),
  );
}

/// 审核状态页：三步进度 + 禁止返回（M1-8）。class GatePendingPage extends StatelessWidget {
  const GatePendingPage({super.key});

  @override
  Widget build(BuildContext context) {
    final steps = <(String, bool)>[
      ('提交邀请码与称呼', true),
      ('管理员人工审核（1–2 天）', false),
      ('进入社群', false),
    ];
    return PopScope(
      canPop: false,
      child: Scaffold(
        appBar: AppBar(title: const Text('审核中'), automaticallyImplyLeading: false),
        body: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            for (final step in steps) _StepRow(title: step.$1, done: step.$2),
            const SizedBox(height: 24),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: WmtColors.greenSoft,
                borderRadius: BorderRadius.circular(18),
              ),
              child: const Text(
                '审核通过后，用手机号短信验证码登录即可进入。\n不会向你推送任何营销信息。',
                style: TextStyle(color: WmtColors.greenDeep, height: 1.6),
              ),
            ),
            const SizedBox(height: 16),
            TextButton(
              onPressed: () => context.go('/login'),
              child: const Text('已通过？去登录'),
            ),
          ],
        ),
      ),
    );
  }
}

class _StepRow extends StatelessWidget {
  const _StepRow({required this.title, required this.done});

  final String title;
  final bool done;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Icon(done ? Icons.check_circle : Icons.radio_button_unchecked,
              size: 20, color: done ? WmtColors.green : WmtColors.divider),
          const SizedBox(width: 12),
          Expanded(
            child: Text(title,
                style: TextStyle(
                    color: done ? WmtColors.ink : WmtColors.skyDeep,
                    fontWeight: done ? FontWeight.bold : FontWeight.normal)),
          ),
        ],
      ),
    );
  }
}
