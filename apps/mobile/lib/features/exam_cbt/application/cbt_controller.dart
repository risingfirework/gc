import 'dart:async';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../../../core/network/api_client.dart';
import '../data/cbt_repository.dart';
import '../domain/cbt_models.dart';

enum AnswerSaveStatus { idle, pending, saving, saved, offline, error }

class CBTState {
  const CBTState({
    this.session,
    this.currentIndex = 0,
    this.answers = const {},
    this.doubtful = const {},
    this.remainingSeconds = 0,
    this.online = true,
    this.loading = true,
    this.submitting = false,
    this.saveStatus = AnswerSaveStatus.idle,
    this.error,
    this.result,
    this.securityEvents = const [],
  });
  final ExamSession? session;
  final int currentIndex;
  final Map<String, String> answers;
  final Set<String> doubtful;
  final int remainingSeconds;
  final bool online;
  final bool loading;
  final bool submitting;
  final AnswerSaveStatus saveStatus;
  final String? error;
  final SubmitResult? result;
  final List<String> securityEvents;

  CBTState copyWith({
    ExamSession? session,
    int? currentIndex,
    Map<String, String>? answers,
    Set<String>? doubtful,
    int? remainingSeconds,
    bool? online,
    bool? loading,
    bool? submitting,
    AnswerSaveStatus? saveStatus,
    String? error,
    bool clearError = false,
    SubmitResult? result,
    List<String>? securityEvents,
  }) =>
      CBTState(
        session: session ?? this.session,
        currentIndex: currentIndex ?? this.currentIndex,
        answers: answers ?? this.answers,
        doubtful: doubtful ?? this.doubtful,
        remainingSeconds: remainingSeconds ?? this.remainingSeconds,
        online: online ?? this.online,
        loading: loading ?? this.loading,
        submitting: submitting ?? this.submitting,
        saveStatus: saveStatus ?? this.saveStatus,
        error: clearError ? null : error ?? this.error,
        result: result ?? this.result,
        securityEvents: securityEvents ?? this.securityEvents,
      );
}

class CBTController extends StateNotifier<CBTState> {
  CBTController(this._examId, this._repository, this._connectivity)
      : super(const CBTState()) {
    _connectivitySubscription =
        _connectivity.onConnectivityChanged.listen(_connectivityChanged);
    unawaited(initialize());
  }

  final String _examId;
  final CBTRepository _repository;
  final Connectivity _connectivity;
  final Map<String, String> _pending = {};
  final Map<String, Timer> _debounces = {};
  final Map<String, Future<bool>> _inFlight = {};
  late final StreamSubscription<List<ConnectivityResult>>
      _connectivitySubscription;
  Timer? _clock;
  Duration _serverOffset = Duration.zero;
  bool _submitStarted = false;
  bool _autoSubmitAttempted = false;

  Future<void> initialize() async {
    try {
      final connectivity = await _connectivity.checkConnectivity();
      final session = await _repository.startExam(_examId);
      _serverOffset = session.serverTime.difference(DateTime.now().toUtc());
      final stored = await SharedPreferences.getInstance();
      final answers = <String, String>{};
      final doubtful = <String>{};
      for (final question in session.questions) {
        final answer =
            stored.getString(_answerKey(session.userExamId, question.id));
        if (answer != null) answers[question.id] = answer;
        if (stored.getBool(_doubtKey(session.userExamId, question.id)) ??
            false) {
          doubtful.add(question.id);
        }
      }
      state = state.copyWith(
          session: session,
          answers: answers,
          doubtful: doubtful,
          online: !connectivity.contains(ConnectivityResult.none),
          loading: false,
          clearError: true);
      _updateClock();
      _clock = Timer.periodic(
          const Duration(milliseconds: 250), (_) => _updateClock());
    } on ApiFailure catch (error) {
      state = state.copyWith(loading: false, error: error.message);
    } catch (_) {
      state =
          state.copyWith(loading: false, error: 'Ujian tidak dapat dimuat.');
    }
  }

  void selectAnswer(String questionId, String selectedOption) {
    final session = state.session;
    if (session == null ||
        state.result != null ||
        state.remainingSeconds <= 0) {
      return;
    }
    state = state.copyWith(
        answers: {...state.answers, questionId: selectedOption},
        saveStatus:
            state.online ? AnswerSaveStatus.pending : AnswerSaveStatus.offline,
        clearError: true);
    _pending[questionId] = selectedOption;
    unawaited(_persistAnswer(session.userExamId, questionId, selectedOption));
    _debounces.remove(questionId)?.cancel();
    _debounces[questionId] = Timer(
        const Duration(milliseconds: 350), () => unawaited(_sync(questionId)));
  }

  Future<bool> _sync(String questionId) {
    final existing = _inFlight[questionId];
    if (existing != null) return existing;
    final session = state.session;
    final option = _pending.remove(questionId);
    if (session == null || option == null) return Future.value(true);
    final operation =
        _performSync(session, questionId, option).whenComplete(() {
      _inFlight.remove(questionId);
      if (_pending.containsKey(questionId)) {
        _debounces[questionId] =
            Timer(Duration.zero, () => unawaited(_sync(questionId)));
      }
    });
    _inFlight[questionId] = operation;
    return operation;
  }

  Future<bool> _performSync(
      ExamSession session, String questionId, String option) async {
    if (!state.online) {
      _pending.putIfAbsent(questionId, () => option);
      return false;
    }
    state = state.copyWith(saveStatus: AnswerSaveStatus.saving);
    try {
      final result =
          await _repository.syncAnswer(session.userExamId, questionId, option);
      if (!_pending.containsKey(questionId)) {
        state = state.copyWith(
            saveStatus: AnswerSaveStatus.saved,
            remainingSeconds: result.remainingSeconds,
            clearError: true);
      }
      return true;
    } on ApiFailure catch (error) {
      _pending.putIfAbsent(questionId, () => option);
      state = state.copyWith(
          saveStatus: error.networkError
              ? AnswerSaveStatus.offline
              : AnswerSaveStatus.error,
          online: error.networkError ? false : state.online,
          error: error.networkError
              ? 'Jawaban ditahan di perangkat sampai koneksi kembali.'
              : error.message);
      return false;
    }
  }

  Future<void> submit({bool automatic = false}) async {
    final session = state.session;
    if (session == null || _submitStarted) return;
    _submitStarted = true;
    state = state.copyWith(submitting: true, clearError: true);
    for (var timer in _debounces.values) {
      timer.cancel();
    }
    _debounces.clear();
    await Future.wait(_inFlight.values.toList());
    if (!state.online) {
      _submitStarted = false;
      state = state.copyWith(
          submitting: false,
          error:
              'Perangkat offline. Jawaban tetap tersimpan lokal; sambungkan internet untuk submit.');
      return;
    }
    await Future.wait(_pending.keys.toList().map(_sync));
    if (_pending.isNotEmpty) {
      _submitStarted = false;
      state = state.copyWith(
          submitting: false,
          error: 'Sebagian jawaban belum tersinkron. Coba submit kembali.');
      return;
    }
    try {
      final result = await _repository.submitExam(session.userExamId);
      await _clearStoredExam(session);
      state =
          state.copyWith(result: result, submitting: false, clearError: true);
    } on ApiFailure catch (error) {
      _submitStarted = false;
      state = state.copyWith(
          submitting: false,
          error: automatic ? 'Waktu habis. ${error.message}' : error.message);
    }
  }

  void previous() {
    if (state.currentIndex > 0) {
      state = state.copyWith(currentIndex: state.currentIndex - 1);
    }
  }

  void next() {
    final count = state.session?.questions.length ?? 0;
    if (state.currentIndex + 1 < count) {
      state = state.copyWith(currentIndex: state.currentIndex + 1);
    }
  }

  void goTo(int index) {
    if (index >= 0 && index < (state.session?.questions.length ?? 0)) {
      state = state.copyWith(currentIndex: index);
    }
  }

  void toggleDoubtful(String questionId) {
    final session = state.session;
    if (session == null) return;
    final next = {...state.doubtful};
    if (!next.remove(questionId)) next.add(questionId);
    state = state.copyWith(doubtful: next);
    unawaited(_persistDoubt(
        session.userExamId, questionId, next.contains(questionId)));
  }

  void recordSecurityEvent(String event) {
    state = state.copyWith(securityEvents: [
      ...state.securityEvents.skip(state.securityEvents.length > 49
          ? state.securityEvents.length - 49
          : 0),
      event
    ]);
  }

  void _updateClock() {
    final session = state.session;
    if (session == null || state.result != null) return;
    final adjustedNow = DateTime.now().toUtc().add(_serverOffset);
    final milliseconds = session.endsAt.difference(adjustedNow).inMilliseconds;
    final seconds = milliseconds <= 0 ? 0 : (milliseconds / 1000).ceil();
    if (seconds != state.remainingSeconds) {
      state = state.copyWith(remainingSeconds: seconds);
    }
    if (seconds == 0 && !_submitStarted && !_autoSubmitAttempted) {
      _autoSubmitAttempted = true;
      unawaited(submit(automatic: true));
    }
  }

  void _connectivityChanged(List<ConnectivityResult> results) {
    final online = !results.contains(ConnectivityResult.none);
    state = state.copyWith(
        online: online,
        saveStatus: online ? state.saveStatus : AnswerSaveStatus.offline);
    if (online) {
      for (final questionId in _pending.keys.toList()) {
        unawaited(_sync(questionId));
      }
      if (state.remainingSeconds == 0 && state.result == null) {
        _autoSubmitAttempted = false;
      }
    }
  }

  Future<void> _persistAnswer(
          String exam, String question, String answer) async =>
      (await SharedPreferences.getInstance())
          .setString(_answerKey(exam, question), answer);
  Future<void> _persistDoubt(String exam, String question, bool value) async =>
      (await SharedPreferences.getInstance())
          .setBool(_doubtKey(exam, question), value);
  String _answerKey(String exam, String question) =>
      'cbt:$exam:answer:$question';
  String _doubtKey(String exam, String question) => 'cbt:$exam:doubt:$question';
  Future<void> _clearStoredExam(ExamSession session) async {
    final storage = await SharedPreferences.getInstance();
    for (final question in session.questions) {
      await storage.remove(_answerKey(session.userExamId, question.id));
      await storage.remove(_doubtKey(session.userExamId, question.id));
    }
  }

  @override
  void dispose() {
    _clock?.cancel();
    for (var timer in _debounces.values) {
      timer.cancel();
    }
    _connectivitySubscription.cancel();
    super.dispose();
  }
}

final cbtRepositoryProvider = Provider<CBTRepository>(
    (ref) => CBTRepository(ref.watch(apiClientProvider)));
final cbtControllerProvider = StateNotifierProvider.autoDispose
    .family<CBTController, CBTState, String>((ref, examId) {
  return CBTController(
      examId, ref.watch(cbtRepositoryProvider), Connectivity());
});
