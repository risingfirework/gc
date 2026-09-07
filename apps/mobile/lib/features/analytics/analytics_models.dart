import '../exam_cbt/domain/cbt_models.dart';

class SubjectResult {
  const SubjectResult(
      {required this.name,
      required this.correct,
      required this.total,
      required this.score});
  final String name;
  final int correct;
  final int total;
  final double score;
  factory SubjectResult.fromJson(Map<String, dynamic> json) => SubjectResult(
      name: json['subject_name'] as String,
      correct: (json['correct_answers'] as num).toInt(),
      total: (json['total_questions'] as num).toInt(),
      score: (json['score'] as num).toDouble());
}

class AnswerReview {
  const AnswerReview(
      {required this.questionId,
      required this.subject,
      required this.content,
      required this.options,
      required this.selectedOption,
      required this.correctAnswer,
      required this.correct,
      required this.explanation,
      required this.videoUrl});
  final String questionId;
  final String subject;
  final String content;
  final List<QuestionOption> options;
  final String? selectedOption;
  final String correctAnswer;
  final bool correct;
  final String explanation;
  final String? videoUrl;
  factory AnswerReview.fromJson(Map<String, dynamic> json) => AnswerReview(
        questionId: json['question_id'] as String,
        subject: json['subject_name'] as String,
        content: json['content_text'] as String,
        options: (json['options'] as List<dynamic>)
            .map(
                (item) => QuestionOption.fromJson(item as Map<String, dynamic>))
            .toList(growable: false),
        selectedOption: json['selected_option'] as String?,
        correctAnswer: json['correct_answer'] as String,
        correct: json['is_correct'] as bool,
        explanation: json['explanation_text'] as String,
        videoUrl: json['explanation_video_url'] as String?,
      );
}

class ExamAnalytics {
  const ExamAnalytics(
      {required this.title,
      required this.totalScore,
      required this.passingScore,
      required this.passed,
      required this.correct,
      required this.wrong,
      required this.unanswered,
      required this.subjects,
      required this.review});
  final String title;
  final double totalScore;
  final double passingScore;
  final bool passed;
  final int correct;
  final int wrong;
  final int unanswered;
  final List<SubjectResult> subjects;
  final List<AnswerReview> review;
  factory ExamAnalytics.fromJson(Map<String, dynamic> json) => ExamAnalytics(
        title: json['title'] as String,
        totalScore: (json['total_score'] as num).toDouble(),
        passingScore: (json['passing_score'] as num).toDouble(),
        passed: json['passed'] as bool,
        correct: (json['correct_answers'] as num).toInt(),
        wrong: (json['wrong_answers'] as num).toInt(),
        unanswered: (json['unanswered'] as num).toInt(),
        subjects: (json['subjects'] as List<dynamic>)
            .map((item) => SubjectResult.fromJson(item as Map<String, dynamic>))
            .toList(growable: false),
        review: (json['review'] as List<dynamic>)
            .map((item) => AnswerReview.fromJson(item as Map<String, dynamic>))
            .toList(growable: false),
      );
}
