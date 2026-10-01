import '../../../core/network/api_client.dart';
import '../domain/cbt_models.dart';

class CBTRepository {
  const CBTRepository(this._api);
  final ApiClient _api;

  Future<ExamSession> startExam(String examId) async =>
      ExamSession.fromJson(await _api.get('/exams/$examId/start'));
  Future<AnswerSyncResult> syncAnswer(
    String userExamId,
    String questionId,
    String selectedOption,
  ) async => AnswerSyncResult.fromJson(
    await _api.post('/cbt/answers/sync', {
      'exam_id': userExamId,
      'question_id': questionId,
      'selected_option': selectedOption,
    }),
  );
  Future<SubmitResult> submitExam(String userExamId) async =>
      SubmitResult.fromJson(await _api.post('/exams/$userExamId/submit'));

  /// Melaporkan siswa keluar aplikasi/tab. Server memutuskan apakah layar
  /// harus dikunci dan hanya berlaku untuk mode CBT.
  Future<ScreenLockState> reportViolation(
    String userExamId,
    String event,
  ) async => ScreenLockState.fromJson(
    await _api.post('/cbt/violations', {
      'user_exam_id': userExamId,
      'event': event,
    }),
  );

  /// Polling status kunci supaya blokir yang dibuka guru langsung terasa.
  Future<ScreenLockState> screenLock(String userExamId) async =>
      ScreenLockState.fromJson(
        await _api.get('/exams/$userExamId/screen-lock'),
      );
}
