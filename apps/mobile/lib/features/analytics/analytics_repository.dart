import '../../core/network/api_client.dart';
import 'analytics_models.dart';

class AnalyticsRepository {
  const AnalyticsRepository(this._api);
  final ApiClient _api;
  Future<ExamAnalytics> getResult(String userExamId) async =>
      ExamAnalytics.fromJson(await _api.get('/exams/$userExamId/result'));
}
