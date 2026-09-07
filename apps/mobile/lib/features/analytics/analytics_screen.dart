import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/network/api_client.dart';
import '../exam_cbt/exam_screen.dart';
import 'analytics_models.dart';
import 'analytics_repository.dart';

class AnalyticsScreen extends ConsumerWidget {
  const AnalyticsScreen({super.key, required this.userExamId});
  final String userExamId;

  @override
  Widget build(BuildContext context, WidgetRef ref) => Scaffold(
        appBar: AppBar(title: const Text('Hasil & Pembahasan')),
        body: FutureBuilder<ExamAnalytics>(
          future: AnalyticsRepository(ref.read(apiClientProvider))
              .getResult(userExamId),
          builder: (context, snapshot) {
            if (snapshot.connectionState == ConnectionState.waiting) {
              return const Center(child: CircularProgressIndicator());
            }
            if (snapshot.hasError) {
              return Center(
                  child: Padding(
                      padding: const EdgeInsets.all(24),
                      child: Text('Hasil gagal dimuat: ${snapshot.error}',
                          textAlign: TextAlign.center)));
            }
            final result = snapshot.data!;
            return ListView(padding: const EdgeInsets.all(18), children: [
              Card(
                  child: Padding(
                      padding: const EdgeInsets.all(24),
                      child: Column(children: [
                        Text(result.title,
                            style: Theme.of(context).textTheme.titleLarge,
                            textAlign: TextAlign.center),
                        const SizedBox(height: 12),
                        Text(result.totalScore.toStringAsFixed(2),
                            style: TextStyle(
                                fontSize: 58,
                                fontWeight: FontWeight.w900,
                                color:
                                    result.passed ? Colors.green : Colors.red)),
                        Text(result.passed ? 'LULUS' : 'TIDAK LULUS',
                            style: const TextStyle(
                                fontSize: 20, fontWeight: FontWeight.bold)),
                        Text(
                            '${result.correct} benar · ${result.wrong} salah · ${result.unanswered} kosong')
                      ]))),
              const SizedBox(height: 16),
              Card(
                  child: Padding(
                      padding: const EdgeInsets.all(20),
                      child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            Text('Performa per mata pelajaran',
                                style: Theme.of(context).textTheme.titleMedium),
                            const SizedBox(height: 16),
                            for (final subject in result.subjects)
                              Padding(
                                  padding: const EdgeInsets.only(bottom: 16),
                                  child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        Row(
                                            mainAxisAlignment:
                                                MainAxisAlignment.spaceBetween,
                                            children: [
                                              Text(subject.name,
                                                  style: const TextStyle(
                                                      fontWeight:
                                                          FontWeight.bold)),
                                              Text(subject.score
                                                  .toStringAsFixed(1))
                                            ]),
                                        const SizedBox(height: 6),
                                        LinearProgressIndicator(
                                            value: subject.score / 100,
                                            minHeight: 10,
                                            borderRadius:
                                                BorderRadius.circular(10)),
                                        const SizedBox(height: 4),
                                        Text(
                                            '${subject.correct}/${subject.total} benar',
                                            style: Theme.of(context)
                                                .textTheme
                                                .labelSmall)
                                      ]))
                          ]))),
              const SizedBox(height: 28),
              Text('Review Pembahasan',
                  style: Theme.of(context).textTheme.headlineSmall),
              const SizedBox(height: 10),
              for (var index = 0; index < result.review.length; index++)
                _ReviewTile(index: index, item: result.review[index]),
            ]);
          },
        ),
      );
}

class _ReviewTile extends StatelessWidget {
  const _ReviewTile({required this.index, required this.item});
  final int index;
  final AnswerReview item;

  @override
  Widget build(BuildContext context) => Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: ExpansionTile(
        leading: CircleAvatar(
            backgroundColor:
                item.correct ? Colors.green.shade100 : Colors.red.shade100,
            child: Text('${index + 1}')),
        title: Text(item.subject),
        subtitle: Text(item.selectedOption == null
            ? 'Tidak dijawab'
            : item.correct
                ? 'Benar'
                : 'Salah'),
        children: [
          Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    RichLatexText(item.content),
                    const SizedBox(height: 12),
                    for (final option in item.options)
                      Padding(
                          padding: const EdgeInsets.only(bottom: 5),
                          child: Row(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                SizedBox(
                                    width: 28,
                                    child: Text(option.key,
                                        style: const TextStyle(
                                            fontWeight: FontWeight.bold))),
                                Expanded(child: RichLatexText(option.content))
                              ])),
                    const SizedBox(height: 12),
                    Text('Jawaban kamu: ${item.selectedOption ?? "—"}'),
                    Text('Kunci jawaban: ${item.correctAnswer}',
                        style: const TextStyle(fontWeight: FontWeight.bold)),
                    const Divider(height: 28),
                    const Text('Pembahasan',
                        style: TextStyle(fontWeight: FontWeight.bold)),
                    const SizedBox(height: 6),
                    RichLatexText(item.explanation.isEmpty
                        ? 'Pembahasan belum tersedia.'
                        : item.explanation),
                    if (item.videoUrl != null)
                      Padding(
                          padding: const EdgeInsets.only(top: 12),
                          child: OutlinedButton.icon(
                              onPressed: () => launchUrl(
                                  Uri.parse(item.videoUrl!),
                                  mode: LaunchMode.inAppBrowserView),
                              icon: const Icon(Icons.play_circle_outline),
                              label: const Text('Video pembahasan')))
                  ]))
        ],
      ));
}
