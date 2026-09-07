import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

class SecurityUtils {
  static const _channel = MethodChannel('id.tkajuara.app/security');
  static Future<void> enableSecureScreen() async {
    if (defaultTargetPlatform == TargetPlatform.android) {
      await _channel.invokeMethod<void>('enableSecureScreen');
    }
  }

  static Future<void> disableSecureScreen() async {
    if (defaultTargetPlatform == TargetPlatform.android) {
      await _channel.invokeMethod<void>('disableSecureScreen');
    }
  }
}

class ExamLifecycleGuard with WidgetsBindingObserver {
  ExamLifecycleGuard(this.onViolation);
  final void Function(String event) onViolation;
  DateTime? _lastReport;

  void attach() => WidgetsBinding.instance.addObserver(this);
  void detach() => WidgetsBinding.instance.removeObserver(this);

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.inactive &&
        state != AppLifecycleState.paused &&
        state != AppLifecycleState.hidden) {
      return;
    }
    final now = DateTime.now().toUtc();
    if (_lastReport != null &&
        now.difference(_lastReport!) < const Duration(seconds: 1)) {
      return;
    }
    _lastReport = now;
    onViolation('app_${state.name}:${now.toIso8601String()}');
  }
}
