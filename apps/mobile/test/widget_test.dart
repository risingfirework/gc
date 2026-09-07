import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:tka_mobile/app.dart';

void main() {
  testWidgets('menampilkan formulir login', (tester) async {
    await tester.pumpWidget(const ProviderScope(child: TkaApp()));
    expect(find.text('Selamat datang'), findsOneWidget);
    expect(find.text('Masuk'), findsOneWidget);
  });
}
