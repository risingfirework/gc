class QuestionOption {
  const QuestionOption({
    required this.key,
    required this.content,
    required this.imageUrl,
  });
  final String key;
  final String content;
  final String imageUrl;
  factory QuestionOption.fromJson(Map<String, dynamic> json) => QuestionOption(
    key: json['key'] as String,
    content: json['content'] as String,
    imageUrl: json['image_url'] as String? ?? '',
  );
}

class ExamQuestion {
  const ExamQuestion({
    required this.id,
    required this.subjectName,
    required this.contentText,
    required this.questionType,
    required this.presentationType,
    required this.groupCode,
    required this.stimulusText,
    required this.questionImageUrl,
    required this.stimulusImageUrl,
    required this.categoryLabels,
    required this.options,
  });
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
        .map((item) => QuestionOption.fromJson(item as Map<String, dynamic>))
        .toList(growable: false),
  );
}

class ExamSession {
  const ExamSession({
    required this.userExamId,
    required this.examId,
    required this.title,
    required this.serverTime,
    required this.endsAt,
    required this.questions,
    this.examType = '',
    this.screenLockEnabled = false,
  });
  final String userExamId;
  final String examId;
  final String title;
  final DateTime serverTime;
  final DateTime endsAt;
  final List<ExamQuestion> questions;

  /// Mode bank soal, misal "cbt" atau "sell". Kosong berarti server lama yang
  /// tidak mengirim field ini.
  final String examType;

  /// Hanya true untuk mode CBT. Mode sell dan mode lain tidak dikunci.
  final bool screenLockEnabled;

  bool get isCBT => examType.toLowerCase() == 'cbt';

  factory ExamSession.fromJson(Map<String, dynamic> json) => ExamSession(
    userExamId: json['user_exam_id'] as String,
    examId: json['exam_id'] as String,
    title: json['title'] as String,
    serverTime: DateTime.parse(json['server_time'] as String).toUtc(),
    endsAt: DateTime.parse(json['ends_at'] as String).toUtc(),
    examType: json['exam_type'] as String? ?? '',
    screenLockEnabled: json['screen_lock_enabled'] as bool? ?? false,
    questions: (json['questions'] as List<dynamic>)
        .map((item) => ExamQuestion.fromJson(item as Map<String, dynamic>))
        .toList(growable: false),
  );
}

/// ScreenLockState adalah status layar kunci yang dikembalikan server saat
/// siswa dilaporkan keluar aplikasi atau berpindah tab.
class ScreenLockState {
  const ScreenLockState({
    this.locked = false,
    this.enforced = false,
    this.lockSeconds = 0,
    this.violationCount = 0,
    this.serverTime,
    this.unlockUntil,
  });

  /// True berarti klien harus menampilkan layar kunci.
  final bool locked;

  /// False untuk mode non-CBT sehingga klien tidak pernah mengunci.
  final bool enforced;
  final int lockSeconds;
  final int violationCount;
  final DateTime? serverTime;
  final DateTime? unlockUntil;

  factory ScreenLockState.fromJson(Map<String, dynamic> json) {
    final unlock = json['unlock_until'] as String?;
    final server = json['server_time'] as String?;
    return ScreenLockState(
      locked: json['locked'] as bool? ?? false,
      enforced: json['enforced'] as bool? ?? false,
      lockSeconds: json['lock_seconds'] as int? ?? 0,
      violationCount: json['violation_count'] as int? ?? 0,
      serverTime: server == null ? null : DateTime.parse(server).toUtc(),
      unlockUntil: unlock == null ? null : DateTime.parse(unlock).toUtc(),
    );
  }
}

class AnswerSyncResult {
  const AnswerSyncResult({required this.remainingSeconds});
  final int remainingSeconds;
  factory AnswerSyncResult.fromJson(Map<String, dynamic> json) =>
      AnswerSyncResult(
        remainingSeconds: (json['remaining_seconds'] as num).toInt(),
      );
}

class SubmitResult {
  const SubmitResult({
    required this.userExamId,
    required this.totalScore,
    required this.passingScore,
    required this.passed,
  });
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
