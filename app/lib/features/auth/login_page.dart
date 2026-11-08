import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../core/network/api_client.dart';
import '../../core/theme/app_theme.dart';
import '../auth/auth_controller.dart';

/// 短信验证码登录页（M1-9）。审核通过后凭手机号进入。
class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _phone = TextEditingController();
  final _code = TextEditingController();
  Timer? _timer;
  int _countdown = 0;
  bool _sending = false;
  bool _loggingIn = false;

  @override
  void dispose() {
    _timer?.cancel();
    _phone.dispose();
    _code.dispose();
    super.dispose();
  }

  void _startCountdown() {
    setState(() => _countdown = 60);
    _timer = Timer.periodic(const Duration(seconds: 1), (t) {
      if (!mounted) return t.cancel();
      setState(() => _countdown--);
      if (_countdown <= 0) t.cancel();
    });
  }

  Future<void> _sendCode() async {
    final phone = _phone.text.trim();
    if (!RegExp(r'^1[3-9]\d{9}$').hasMatch(phone)) {
      _toast('请填写正确的手机号');
      return;
    }
    setState(() => _sending = true);
    try {
      await context.read<AuthController>().sendCode(phone);
      if (mounted) {
        _startCountdown();
        _toast('验证码已发送');
      }
    } on AppError catch (e) {
      _toast(e.message);
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  Future<void> _login() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _loggingIn = true);
    try {
      final res = await context
          .read<AuthController>()
          .login(_phone.text.trim(), _code.text.trim());
      if (!mounted) return;
      if (res.isMember) {
        context.go('/home'); // 平级重置：唯一允许 go() 的场景之一（架构设计 4.3）
      } else if (res.isPending || res.isRejected) {
        context.push('/gate/pending');
      } else {
        _toast('这个手机号还没有申请记录，请先用邀请码进入');
      }
    } on AppError catch (e) {
      _toast(e.message);
    } finally {
      if (mounted) setState(() => _loggingIn = false);
    }
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('短信登录')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            TextFormField(
              controller: _phone,
              keyboardType: TextInputType.phone,
              decoration: const InputDecoration(labelText: '手机号'),
              validator: (v) =>
                  RegExp(r'^1[3-9]\d{9}$').hasMatch((v ?? '').trim())
                      ? null
                      : '请填写正确的手机号',
            ),
            const SizedBox(height: 16),
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: TextFormField(
                    controller: _code,
                    keyboardType: TextInputType.number,
                    maxLength: 6,
                    decoration: const InputDecoration(
                        labelText: '验证码', counterText: ''),
                    validator: (v) =>
                        (v == null || v.trim().length != 6) ? '6 位验证码' : null,
                  ),
                ),
                const SizedBox(width: 12),
                SizedBox(
                  height: 52,
                  child: OutlinedButton(
                    onPressed: (_sending || _countdown > 0) ? null : _sendCode,
                    child: Text(_countdown > 0 ? '$_countdown s' : '发送验证码'),
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
                onPressed: _loggingIn ? null : _login,
                child: _loggingIn
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                    : const Text('登录'),
              ),
            ),
            const SizedBox(height: 12),
            const Text(
              '只对邀请开放的社群：还没有申请？先用邀请码进入。',
              style: TextStyle(fontSize: 12, color: WmtColors.skyDeep),
            ),
          ],
        ),
      ),
    );
  }
}
