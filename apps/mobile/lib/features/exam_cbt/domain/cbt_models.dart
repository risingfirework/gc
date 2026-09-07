class QuestionOption {
  const QuestionOption(
      {required this.key, required this.content, required this.imageUrl});
  final String key;
  final String content;
  final String imageUrl;
  factory QuestionOption.fromJson(Map<String, dynamic> json) => QuestionOption(
      key: json['key'] as String,
      content: json['content'] as String,
      imageUrl: json['image_url'] as String? ?? '');
}

class ExamQuestion {
  const ExamQuestion(
      {required this.id,
      required this.subjectName,
      required this.contentText,
      required this.questionType,
      required this.presentationType,
      required this.groupCode,
      required this.stimulusText,
      required this.questionImageUrl,
      required this.stimulusImageUrl,
      required this.categoryLabels,
      required this.options});
  final String id;
  final String subjectName;
  final String contentText;
  final String questionType;
  final String presentationType;
  final String groupCode;
  final String stimulusText;
  final String questionImageUrl;
  final String stimulusImageUrl;
  final List<String> categoryLabels;
  final List<QuestionOption> options;
  factory ExamQuestion.fromJson(Map<String, dynamic> json) => ExamQuestion(
        id: json['id'] as String,
        subjectName: json['subject_name'] as String,
        contentText: json['content_text'] as String,
        questionType: json['question_type'] as String? ?? 'single_choice',
        presentationType: json['presentation_type'] as String? ?? 'single',
        groupCode: json['group_code'] as String? ?? '',
        stimulusText: json['stimulus_text'] as String? ?? '',
        questionImageUrl: json['question_image_url'] as String? ?? '',
        stimulusImageUrl: json['stimulus_image_url'] as String? ?? '',
        categoryLabels: (json['category_labels'] as List<dynamic>? ?? const [])
            .map((item) => item as String)
            .toList(growable: false),
        options: (json['options'] as List<dynamic>)
            .map(
                (item) => QuestionOption.fromJson(item as Map<String, dynamic>))
            .toList(growable: false),
      );
}

class ExamSession {
  const ExamSession(
      {required this.userExamId,
      required this.examId,
      required this.title,
      required this.serverTime,
      required this.endsAt,
      required this.questions});
  final String userExamId;
  final String examId;
  final String title;
  final DateTime serverTime;
  final DateTime endsAt;
  final List<ExamQuestion> questions;
  factory ExamSession.fromJson(Map<String, dynamic> json) => ExamSession(
        userExamId: json['user_exam_id'] as String,
        examId: json['exam_id'] as String,
        title: json['title'] as String,
        serverTime: DateTime.parse(json['server_time'] as String).toUtc(),
        endsAt: DateTime.parse(json['ends_at'] as String).toUtc(),
        questions: (json['questions'] as List<dynamic>)
            .map((item) => ExamQuestion.fromJson(item as Map<String, dynamic>))
            .toList(growable: false),
      );
}

class AnswerSyncResult {
  const AnswerSyncResult({required this.remainingSeconds});
  final int remainingSeconds;
  factory AnswerSyncResult.fromJson(Map<String, dynamic> json) =>
      AnswerSyncResult(
          remainingSeconds: (json['remaining_seconds'] as num).toInt());
}

class SubmitResult {
  const SubmitResult(
      {required this.userExamId,
      required this.totalScore,
      required this.passingScore,
      required this.passed});
  final String userExamId;
  final double totalScore;
  final double passingScore;
  final bool passed;
  factory SubmitResult.fromJson(Map<String, dynamic> json) => SubmitResult(
        userExamId: json['user_exam_id'] as String,
        totalScore: (json['total_score'] as num).toDouble(),
        passingScore: (json['passing_score'] as num).toDouble(),
        passed: json['passed'] as bool,
      );
}
