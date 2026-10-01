import 'package:flutter_test/flutter_test.dart';
import 'package:tka_mobile/features/exam_cbt/application/cbt_controller.dart';
import 'package:tka_mobile/features/exam_cbt/domain/cbt_models.dart';

ExamSession _session({
  String examType = 'cbt',
  bool screenLockEnabled = true,
}) {
  final now = DateTime.utc(2026, 1, 1, 10);
  return ExamSession(
    userExamId: 'attempt-1',
    examId: 'exam-1',
    title: 'Ujian',
    serverTime: now,
    endsAt: now.add(const Duration(hours: 1)),
    questions: const [],
    examType: examType,
    screenLockEnabled: screenLockEnabled,
  );
}

void main() {
  group('ExamSession', () {
    test('hanya mode cbt yang dikenali sebagai bank soal CBT', () {
      expect(_session(examType: 'cbt').isCBT, isTrue);
      expect(_session(examType: 'CBT').isCBT, isTrue);
      expect(_session(examType: 'sell').isCBT, isFalse);
      expect(_session(examType: '').isCBT, isFalse);
    });

    test('membaca exam_type dan screen_lock_enabled dari server', () {
      final parsed = ExamSession.fromJson({
        'user_exam_id': 'attempt-1',
        'exam_id': 'exam-1',
        'title': 'Ujian',
        'server_time': '2026-01-01T10:00:00Z',
        'ends_at': '2026-01-01T11:00:00Z',
        'questions': <Map<String, dynamic>>[],
        'exam_type': 'cbt',
        'screen_lock_enabled': true,
      });

      expect(parsed.examType, 'cbt');
      expect(parsed.screenLockEnabled, isTrue);
      expect(parsed.isCBT, isTrue);
    });

    test('server lama tanpa field lock dianggap mode non-CBT', () {
      final parsed = ExamSession.fromJson({
        'user_exam_id': 'attempt-1',
        'exam_id': 'exam-1',
        'title': 'Ujian',
        'server_time': '2026-01-01T10:00:00Z',
        'ends_at': '2026-01-01T11:00:00Z',
        'questions': <Map<String, dynamic>>[],
      });

      expect(parsed.examType, '');
      expect(parsed.screenLockEnabled, isFalse);
      expect(parsed.isCBT, isFalse);
    });
  });

  group('CBTState.screenLockEligible', () {
    test('hanya true untuk sesi CBT dengan flag aktif', () {
      expect(
        const CBTState(session: null).screenLockEligible,
        isFalse,
        reason: 'sesi belum termuat',
      );
      expect(
        CBTState(session: _session()).screenLockEligible,
        isTrue,
      );
    });

    test('mode sell tidak pernah eligible meski flag server aktif', () {
      expect(
        CBTState(
          session: _session(examType: 'sell', screenLockEnabled: false),
        ).screenLockEligible,
        isFalse,
      );
    });

    test('CBT tanpa flag server tetap tidak eligible', () {
      expect(
        CBTState(
          session: _session(examType: 'cbt', screenLockEnabled: false),
        ).screenLockEligible,
        isFalse,
      );
    });
  });

  group('ScreenLockState', () {
    test('membaca status kunci dari server', () {
      final lock = ScreenLockState.fromJson({
        'user_exam_id': 'attempt-1',
        'locked': true,
        'enforced': true,
        'lock_seconds': 5,
        'violation_count': 3,
        'unlock_until': '2026-01-01T10:00:05Z',
        'server_time': '2026-01-01T10:00:00Z',
      });

      expect(lock.locked, isTrue);
      expect(lock.enforced, isTrue);
      expect(lock.lockSeconds, 5);
      expect(lock.violationCount, 3);
      expect(lock.serverTime, DateTime.utc(2026, 1, 1, 10));
      expect(lock.unlockUntil, DateTime.utc(2026, 1, 1, 10, 0, 5));
    });

    test('balasan mode sell selalu tidak ditegakkan', () {
      final lock = ScreenLockState.fromJson({
        'user_exam_id': 'attempt-1',
        'locked': false,
        'enforced': false,
        'server_time': '2026-01-01T10:00:00Z',
      });

      expect(lock.enforced, isFalse);
      expect(lock.locked, isFalse);
      expect(lock.lockSeconds, 0);
      expect(lock.unlockUntil, isNull);
    });
  });

  test('durasi cadangan layar kunci tetap 5 detik', () {
    expect(defaultScreenLockSeconds, 5);
  });
}