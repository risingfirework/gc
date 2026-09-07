import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_math_fork/flutter_math.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/utils/security_utils.dart';
import '../analytics/analytics_screen.dart';
import 'application/cbt_controller.dart';
import 'domain/cbt_models.dart';

class ExamScreen extends ConsumerStatefulWidget {
  const ExamScreen({super.key, required this.examId});
  final String examId;
  @override
  ConsumerState<ExamScreen> createState() => _ExamScreenState();
}

class _ExamScreenState extends ConsumerState<ExamScreen> {
  late final ExamLifecycleGuard lifecycleGuard;

  @override
  void initState() {
    super.initState();
    unawaited(SecurityUtils.enableSecureScreen());
    lifecycleGuard = ExamLifecycleGuard((event) {
      if (mounted) {
        ref
            .read(cbtControllerProvider(widget.examId).notifier)
            .recordSecurityEvent(event);
      }
    })
      ..attach();
  }

  @override
  void dispose() {
    lifecycleGuard.detach();
    unawaited(SecurityUtils.disableSecureScreen());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(cbtControllerProvider(widget.examId));
    final controller = ref.read(cbtControllerProvider(widget.examId).notifier);
    if (state.loading) {
      return const Scaffold(
          body: Center(
              child: Column(mainAxisSize: MainAxisSize.min, children: [
        CircularProgressIndicator(),
        SizedBox(height: 16),
        Text('Menyiapkan ujian dan menyinkronkan waktu...')
      ])));
    }
    if (state.result != null) return _ResultView(result: state.result!);
    final session = state.session;
    if (session == null) {
      return Scaffold(
          appBar: AppBar(),
          body: Center(
              child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text(state.error ?? 'Ujian tidak tersedia.',
                      textAlign: TextAlign.center))));
    }
    if (session.questions.isEmpty) {
      return Scaffold(
          appBar: AppBar(title: Text(session.title)),
          body: const Center(child: Text('Soal belum tersedia.')));
    }
    final question = session.questions[state.currentIndex];
    return Scaffold(
      appBar: AppBar(
        title: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(session.title, overflow: TextOverflow.ellipsis),
          Text(
              'Soal ${state.currentIndex + 1} dari ${session.questions.length}',
              style: Theme.of(context).textTheme.labelSmall)
        ]),
        actions: [
          Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: _Timer(seconds: state.remainingSeconds))
        ],
      ),
      body: SafeArea(
          child: Column(children: [
        _ConnectionStatus(online: state.online, saveStatus: state.saveStatus),
        if (state.error != null)
          _ErrorBanner(
              message: state.error!,
              retry: state.remainingSeconds == 0
                  ? () => controller.submit()
                  : null),
        Expanded(child: LayoutBuilder(builder: (context, constraints) {
          final questionCard = _QuestionCard(
              question: question,
              selected: state.answers[question.id],
              doubtful: state.doubtful.contains(question.id),
              onSelect: (option) =>
                  controller.selectAnswer(question.id, option),
              onDoubtful: () => controller.toggleDoubtful(question.id),
              previous: state.currentIndex > 0 ? controller.previous : null,
              next: state.currentIndex + 1 < session.questions.length
                  ? controller.next
                  : null,
              submit:
                  state.submitting ? null : () => _confirmSubmit(controller));
          final navigator = _QuestionNavigator(
              session: session,
              state: state,
              onSelect: controller.goTo,
              onSubmit:
                  state.submitting ? null : () => _confirmSubmit(controller));
          if (constraints.maxWidth >= 820) {
            return Padding(
                padding: const EdgeInsets.all(20),
                child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(child: questionCard),
                      const SizedBox(width: 16),
                      SizedBox(width: 280, child: navigator)
                    ]));
          }
          return ListView(
              padding: const EdgeInsets.all(16),
              children: [navigator, const SizedBox(height: 14), questionCard]);
        })),
      ])),
    );
  }

  Future<void> _confirmSubmit(CBTController controller) async {
    final confirmed = await showDialog<bool>(
            context: context,
            builder: (context) => AlertDialog(
                    title: const Text('Submit ujian?'),
                    content: const Text(
                        'Pastikan semua jawaban telah diperiksa. Ujian tidak dapat dilanjutkan setelah submit.'),
                    actions: [
                      TextButton(
                          onPressed: () => Navigator.pop(context, false),
                          child: const Text('Batal')),
                      FilledButton(
                          onPressed: () => Navigator.pop(context, true),
                          child: const Text('Submit'))
                    ])) ??
        false;
    if (confirmed) await controller.submit();
  }
}

class _QuestionCard extends StatelessWidget {
  const _QuestionCard(
      {required this.question,
      required this.selected,
      required this.doubtful,
      required this.onSelect,
      required this.onDoubtful,
      this.previous,
      this.next,
      this.submit});
  final ExamQuestion question;
  final String? selected;
  final bool doubtful;
  final ValueChanged<String> onSelect;
  final VoidCallback onDoubtful;
  final VoidCallback? previous;
  final VoidCallback? next;
  final VoidCallback? submit;

  @override
  Widget build(BuildContext context) => Card(
      child: Padding(
          padding: const EdgeInsets.all(20),
          child:
              Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            Text(question.subjectName.toUpperCase(),
                style: TextStyle(
                    color: Theme.of(context).colorScheme.primary,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 1)),
            if (question.presentationType == 'group') ...[
              const SizedBox(height: 14),
              Container(
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                      color: Theme.of(context).colorScheme.surfaceContainer,
                      borderRadius: BorderRadius.circular(12),
                      border: Border(
                          left: BorderSide(
                              width: 4,
                              color: Theme.of(context).colorScheme.primary))),
                  child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('STIMULUS ${question.groupCode}',
                            style: TextStyle(
                                color: Theme.of(context).colorScheme.primary,
                                fontWeight: FontWeight.w700)),
                        const SizedBox(height: 8),
                        RichLatexText(question.stimulusText),
                        _QuestionMedia(
                            source: question.stimulusImageUrl,
                            semanticLabel: 'Gambar stimulus soal')
                      ]))
            ],
            const SizedBox(height: 20),
            RichLatexText(question.contentText,
                style: Theme.of(context).textTheme.titleLarge),
            _QuestionMedia(
                source: question.questionImageUrl,
                semanticLabel: 'Gambar pertanyaan'),
            const SizedBox(height: 22),
            if (question.questionType == 'multiple_choice')
              const Padding(
                  padding: EdgeInsets.only(bottom: 12),
                  child: Text('Pilih semua jawaban yang benar.')),
            if (question.questionType != 'category')
              ...question.options.map((option) => Padding(
                  padding: const EdgeInsets.only(bottom: 10),
                  child: _OptionTile(
                      option: option,
                      selected: question.questionType == 'multiple_choice'
                          ? (selected ?? '').split(',').contains(option.key)
                          : selected == option.key,
                      onTap: () {
                        if (question.questionType != 'multiple_choice') {
                          onSelect(option.key);
                          return;
                        }
                        final values = (selected ?? '')
                            .split(',')
                            .where((value) => value.isNotEmpty)
                            .toSet();
                        values.contains(option.key)
                            ? values.remove(option.key)
                            : values.add(option.key);
                        if (values.isEmpty) return;
                        final sorted = values.toList()..sort();
                        onSelect(sorted.join(','));
                      },
                      imageUrl: option.imageUrl))),
            if (question.questionType == 'category')
              ...question.options.map((option) {
                final values = <String, String>{};
                for (final part in (selected ?? '').split(';')) {
                  final separator = part.indexOf('=');
                  if (separator > 0) {
                    values[part.substring(0, separator)] =
                        part.substring(separator + 1);
                  }
                }
                return Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Row(children: [
                      Expanded(
                          child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                            RichLatexText('${option.key}. ${option.content}'),
                            _QuestionMedia(
                                source: option.imageUrl,
                                semanticLabel:
                                    'Gambar pernyataan ${option.key}')
                          ])),
                      const SizedBox(width: 12),
                      DropdownButton<String>(
                          hint: const Text('Kategori'),
                          value: values[option.key],
                          items: question.categoryLabels
                              .map((label) => DropdownMenuItem(
                                  value: label, child: Text(label)))
                              .toList(growable: false),
                          onChanged: (label) {
                            if (label == null) return;
                            values[option.key] = label;
                            final keys = values.keys.toList()..sort();
                            onSelect(keys
                                .map((key) => '$key=${values[key]}')
                                .join(';'));
                          })
                    ]));
              }),
            const SizedBox(height: 14),
            Wrap(
                alignment: WrapAlignment.spaceBetween,
                runSpacing: 10,
                spacing: 10,
                children: [
                  OutlinedButton.icon(
                      onPressed: previous,
                      icon: const Icon(Icons.chevron_left),
                      label: const Text('Sebelumnya')),
                  FilledButton.tonalIcon(
                      onPressed: onDoubtful,
                      style: doubtful
                          ? FilledButton.styleFrom(
                              backgroundColor: Colors.amber.shade200)
                          : null,
                      icon: const Icon(Icons.flag_outlined),
                      label: Text(doubtful ? 'Batalkan ragu' : 'Ragu-ragu')),
                  next != null
                      ? FilledButton.icon(
                          onPressed: next,
                          iconAlignment: IconAlignment.end,
                          icon: const Icon(Icons.chevron_right),
                          label: const Text('Selanjutnya'))
                      : FilledButton.icon(
                          onPressed: submit,
                          icon: const Icon(Icons.check_circle_outline),
                          label: const Text('Submit ujian')),
                ]),
          ])));
}

class _OptionTile extends StatelessWidget {
  const _OptionTile(
      {required this.option,
      required this.selected,
      required this.onTap,
      required this.imageUrl});
  final QuestionOption option;
  final bool selected;
  final VoidCallback onTap;
  final String imageUrl;
  @override
  Widget build(BuildContext context) => Material(
      color: selected
          ? Theme.of(context).colorScheme.primaryContainer
          : Theme.of(context).colorScheme.surface,
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: BorderSide(
              color: selected
                  ? Theme.of(context).colorScheme.primary
                  : Theme.of(context).dividerColor)),
      child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: onTap,
          child: Padding(
              padding: const EdgeInsets.all(14),
              child: Row(children: [
                CircleAvatar(
                    radius: 17,
                    backgroundColor: selected
                        ? Theme.of(context).colorScheme.primary
                        : Theme.of(context).colorScheme.surfaceContainerHighest,
                    foregroundColor: selected ? Colors.white : null,
                    child: Text(option.key,
                        style: const TextStyle(fontWeight: FontWeight.bold))),
                const SizedBox(width: 12),
                Expanded(
                    child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                      RichLatexText(option.content),
                      _QuestionMedia(
                          source: imageUrl,
                          semanticLabel: 'Gambar pilihan ${option.key}')
                    ]))
              ]))));
}

class _QuestionMedia extends StatelessWidget {
  const _QuestionMedia({required this.source, required this.semanticLabel});
  final String source;
  final String semanticLabel;

  @override
  Widget build(BuildContext context) {
    if (source.isEmpty) return const SizedBox.shrink();
    Widget image;
    if (source.startsWith('data:image/')) {
      try {
        image = Image.memory(
            base64Decode(source.substring(source.indexOf(',') + 1)),
            fit: BoxFit.contain,
            semanticLabel: semanticLabel);
      } on FormatException {
        return const SizedBox.shrink();
      }
    } else {
      image = Image.network(source,
          fit: BoxFit.contain,
          semanticLabel: semanticLabel,
          errorBuilder: (_, __, ___) => const SizedBox.shrink());
    }
    return Padding(
        padding: const EdgeInsets.only(top: 12),
        child:
            ClipRRect(borderRadius: BorderRadius.circular(12), child: image));
  }
}

class _QuestionNavigator extends StatelessWidget {
  const _QuestionNavigator(
      {required this.session,
      required this.state,
      required this.onSelect,
      required this.onSubmit});
  final ExamSession session;
  final CBTState state;
  final ValueChanged<int> onSelect;
  final VoidCallback? onSubmit;
  @override
  Widget build(BuildContext context) => Card(
      child: Padding(
          padding: const EdgeInsets.all(16),
          child:
              Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            Text(
                '${state.answers.length}/${session.questions.length} soal dijawab',
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 14),
            GridView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
                    maxCrossAxisExtent: 48,
                    mainAxisSpacing: 8,
                    crossAxisSpacing: 8),
                itemCount: session.questions.length,
                itemBuilder: (context, index) {
                  final id = session.questions[index].id;
                  final isDoubtful = state.doubtful.contains(id);
                  final answered = state.answers.containsKey(id);
                  final color = isDoubtful
                      ? Colors.amber.shade200
                      : answered
                          ? Theme.of(context).colorScheme.primaryContainer
                          : Theme.of(context)
                              .colorScheme
                              .surfaceContainerHighest;
                  return Material(
                      color: color,
                      shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(8),
                          side: index == state.currentIndex
                              ? BorderSide(
                                  color: Theme.of(context).colorScheme.primary,
                                  width: 2)
                              : BorderSide.none),
                      child: InkWell(
                          borderRadius: BorderRadius.circular(8),
                          onTap: () => onSelect(index),
                          child: Center(
                              child: Text('${index + 1}',
                                  style: const TextStyle(
                                      fontWeight: FontWeight.bold)))));
                }),
            const SizedBox(height: 14),
            Text(
                '${state.securityEvents.length} aktivitas keluar aplikasi tercatat',
                style: Theme.of(context).textTheme.labelSmall),
            const SizedBox(height: 12),
            FilledButton.icon(
                onPressed: onSubmit,
                icon: const Icon(Icons.check_circle_outline),
                label: Text(state.submitting ? 'Mengirim...' : 'Submit ujian')),
          ])));
}

class RichLatexText extends StatelessWidget {
  const RichLatexText(this.content, {super.key, this.style});
  final String content;
  final TextStyle? style;
  @override
  Widget build(BuildContext context) {
    final expression = RegExp(r'\$([^$]+)\$');
    final widgets = <Widget>[];
    var cursor = 0;
    for (final match in expression.allMatches(content)) {
      if (match.start > cursor) {
        widgets.add(Text(content.substring(cursor, match.start), style: style));
      }
      widgets.add(Math.tex(match.group(1)!,
          textStyle: style, mathStyle: MathStyle.text));
      cursor = match.end;
    }
    if (cursor < content.length) {
      widgets.add(Text(content.substring(cursor), style: style));
    }
    return Wrap(
        crossAxisAlignment: WrapCrossAlignment.center,
        spacing: 3,
        runSpacing: 6,
        children: widgets.isEmpty ? [Text(content, style: style)] : widgets);
  }
}

class _Timer extends StatelessWidget {
  const _Timer({required this.seconds});
  final int seconds;
  @override
  Widget build(BuildContext context) {
    final hours = seconds ~/ 3600;
    final minutes = (seconds % 3600) ~/ 60;
    final rest = seconds % 60;
    final value = hours > 0
        ? '$hours:${minutes.toString().padLeft(2, '0')}:${rest.toString().padLeft(2, '0')}'
        : '${minutes.toString().padLeft(2, '0')}:${rest.toString().padLeft(2, '0')}';
    return Column(mainAxisAlignment: MainAxisAlignment.center, children: [
      Text('Sisa waktu', style: Theme.of(context).textTheme.labelSmall),
      Text(value,
          style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w800,
              color: seconds < 300
                  ? Colors.red
                  : Theme.of(context).colorScheme.primary))
    ]);
  }
}

class _ConnectionStatus extends StatelessWidget {
  const _ConnectionStatus({required this.online, required this.saveStatus});
  final bool online;
  final AnswerSaveStatus saveStatus;
  @override
  Widget build(BuildContext context) {
    final message = !online
        ? 'Offline — jawaban ditahan di perangkat'
        : switch (saveStatus) {
            AnswerSaveStatus.pending => 'Menunggu autosave...',
            AnswerSaveStatus.saving => 'Menyimpan jawaban...',
            AnswerSaveStatus.saved => 'Jawaban tersimpan',
            AnswerSaveStatus.error => 'Autosave gagal',
            _ => ''
          };
    if (message.isEmpty) return const SizedBox.shrink();
    return Container(
        width: double.infinity,
        color: online ? Colors.green.shade50 : Colors.orange.shade100,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 7),
        child: Text(message,
            textAlign: TextAlign.center,
            style: TextStyle(
                color: online ? Colors.green.shade800 : Colors.orange.shade900,
                fontSize: 12)));
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message, this.retry});
  final String message;
  final VoidCallback? retry;
  @override
  Widget build(BuildContext context) => Material(
      color: Theme.of(context).colorScheme.errorContainer,
      child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: Row(children: [
            Expanded(child: Text(message)),
            if (retry != null)
              TextButton(onPressed: retry, child: const Text('Coba lagi'))
          ])));
}

class _ResultView extends StatelessWidget {
  const _ResultView({required this.result});
  final SubmitResult result;
  @override
  Widget build(BuildContext context) => Scaffold(
      body: SafeArea(
          child: Center(
              child: Card(
                  child: Padding(
                      padding: const EdgeInsets.all(32),
                      child: Column(mainAxisSize: MainAxisSize.min, children: [
                        Text('HASIL UJIAN',
                            style: Theme.of(context).textTheme.labelLarge),
                        const SizedBox(height: 12),
                        Text(result.totalScore.toStringAsFixed(2),
                            style: TextStyle(
                                fontSize: 64,
                                fontWeight: FontWeight.w900,
                                color: Theme.of(context).colorScheme.primary)),
                        Text(result.passed ? 'LULUS' : 'TIDAK LULUS',
                            style: TextStyle(
                                fontSize: 22,
                                fontWeight: FontWeight.bold,
                                color:
                                    result.passed ? Colors.green : Colors.red)),
                        Text('Passing score: ${result.passingScore}'),
                        const SizedBox(height: 24),
                        FilledButton(
                            onPressed: () => Navigator.pushReplacement(
                                context,
                                MaterialPageRoute(
                                    builder: (_) => AnalyticsScreen(
                                        userExamId: result.userExamId))),
                            child: const Text('Lihat analitik & pembahasan')),
                        const SizedBox(height: 8),
                        TextButton(
                            onPressed: () => Navigator.pop(context),
                            child: const Text('Kembali'))
                      ]))))));
}
