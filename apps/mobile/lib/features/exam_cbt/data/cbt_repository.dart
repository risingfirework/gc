import '../../../core/network/api_client.dart';
import '../domain/cbt_models.dart';

class CBTRepository {
  const CBTRepository(this._api);
  final ApiClient _api;

  Future<ExamSession> startExam(String examId) async =>
      ExamSession.fromJson(await _api.get('/exams/$examId/start'));
  Future<AnswerSyncResult> syncAnswer(
          String userExamId, String questionId, String selectedOption) async =>
      AnswerSyncResult.fromJson(await _api.post('/cbt/answers/sync', {
        'exam_id': userExamId,
        'question_id': questionId,
        'selected_option': selectedOption,
      }));
  Future<SubmitResult> submitExam(String userExamId) async =>
      SubmitResult.fromJson(await _api.post('/exams/$userExamId/submit'));
}
