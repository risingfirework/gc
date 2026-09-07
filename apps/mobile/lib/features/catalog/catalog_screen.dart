import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/network/api_client.dart';
import '../exam_cbt/exam_screen.dart';
import 'catalog_models.dart';
import 'catalog_repository.dart';

class CatalogScreen extends ConsumerStatefulWidget {
  const CatalogScreen({super.key});
  @override
  ConsumerState<CatalogScreen> createState() => _CatalogScreenState();
}

class _CatalogScreenState extends ConsumerState<CatalogScreen> {
  static const examId = String.fromEnvironment('DEMO_EXAM_ID',
      defaultValue: '00000000-0000-0000-0000-000000000000');
  late Future<List<TryoutPackage>> packages;
  String? checkoutPackageId;

  @override
  void initState() {
    super.initState();
    packages = CatalogRepository(ref.read(apiClientProvider)).listPackages();
  }

  Future<void> _checkout(TryoutPackage item) async {
    final method = await showModalBottomSheet<String>(
        context: context,
        builder: (context) => SafeArea(
            child: Padding(
                padding: const EdgeInsets.all(20),
                child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Text('Bayar ${item.title}',
                          style: Theme.of(context).textTheme.titleLarge),
                      const SizedBox(height: 12),
                      for (final option in const [
                        ('qris', 'QRIS'),
                        ('virtual_account', 'Virtual Account'),
                        ('e_wallet', 'E-Wallet')
                      ])
                        ListTile(
                            title: Text(option.$2),
                            trailing: const Icon(Icons.chevron_right),
                            onTap: () => Navigator.pop(context, option.$1))
                    ]))));
    if (method == null) return;
    setState(() => checkoutPackageId = item.id);
    try {
      final result = await CatalogRepository(ref.read(apiClientProvider))
          .checkout(item.id, method);
      final launched = await launchUrl(Uri.parse(result.paymentUrl),
          mode: LaunchMode.inAppBrowserView);
      if (!launched) {
        throw const ApiFailure('Halaman pembayaran tidak dapat dibuka.');
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text('Invoice ${result.invoiceNumber} berhasil dibuat.')));
      }
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text(error.toString()), backgroundColor: Colors.red));
      }
    } finally {
      if (mounted) setState(() => checkoutPackageId = null);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('TKA Juara')),
        body: RefreshIndicator(
          onRefresh: () async {
            setState(() => packages =
                CatalogRepository(ref.read(apiClientProvider)).listPackages());
            await packages;
          },
          child: ListView(padding: const EdgeInsets.all(20), children: [
            Text('Paket Belajar Saya',
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 14),
            Card(
                child: Padding(
                    padding: const EdgeInsets.all(20),
                    child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('Simulasi TKA',
                              style: TextStyle(
                                  fontSize: 18, fontWeight: FontWeight.bold)),
                          const Text(
                              'Timer server · Autosave aktif · Mode aman'),
                          const SizedBox(height: 16),
                          FilledButton(
                              onPressed: () => Navigator.push(
                                  context,
                                  MaterialPageRoute(
                                      builder: (_) =>
                                          const ExamScreen(examId: examId))),
                              child: const Text('Mulai ujian'))
                        ]))),
            const SizedBox(height: 32),
            Text('Katalog Tryout',
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 14),
            FutureBuilder<List<TryoutPackage>>(
                future: packages,
                builder: (context, snapshot) {
                  if (snapshot.connectionState == ConnectionState.waiting) {
                    return const Center(
                        child: Padding(
                            padding: EdgeInsets.all(32),
                            child: CircularProgressIndicator()));
                  }
                  if (snapshot.hasError) {
                    return Center(
                        child: Text('Katalog gagal dimuat: ${snapshot.error}'));
                  }
                  return Column(children: [
                    for (final item in snapshot.data ?? const [])
                      Padding(
                          padding: const EdgeInsets.only(bottom: 14),
                          child: Card(
                              child: Padding(
                                  padding: const EdgeInsets.all(20),
                                  child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.stretch,
                                      children: [
                                        Text(item.title,
                                            style: Theme.of(context)
                                                .textTheme
                                                .titleLarge),
                                        const SizedBox(height: 6),
                                        Text(item.description),
                                        const SizedBox(height: 12),
                                        Text(
                                            'Rp ${item.price.toStringAsFixed(0)} · Aktif ${item.validityDays} hari',
                                            style: const TextStyle(
                                                fontWeight: FontWeight.bold)),
                                        const SizedBox(height: 14),
                                        FilledButton.icon(
                                            onPressed: checkoutPackageId == null
                                                ? () => _checkout(item)
                                                : null,
                                            icon: const Icon(
                                                Icons.shopping_cart_checkout),
                                            label: Text(
                                                checkoutPackageId == item.id
                                                    ? 'Membuat invoice...'
                                                    : 'Beli paket'))
                                      ]))))
                  ]);
                }),
          ]),
        ),
      );
}
