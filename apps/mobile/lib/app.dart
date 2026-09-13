import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'core/network/api_client.dart';
import 'features/auth/login_screen.dart';

class TkaApp extends ConsumerStatefulWidget {
  const TkaApp({super.key});

  @override
  ConsumerState<TkaApp> createState() => _TkaAppState();
}

class _TkaAppState extends ConsumerState<TkaApp> {
  final navigatorKey = GlobalKey<NavigatorState>();
  StreamSubscription<String>? sessionSubscription;

  @override
  void initState() {
    super.initState();
    sessionSubscription =
        ref.read(apiClientProvider).sessionExpired.listen((message) {
      navigatorKey.currentState?.pushAndRemoveUntil(
          MaterialPageRoute(builder: (_) => const LoginScreen()), (_) => false);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        final context = navigatorKey.currentContext;
        if (context != null) {
          ScaffoldMessenger.of(context)
              .showSnackBar(SnackBar(content: Text(message)));
        }
      });
    });
  }

  @override
  void dispose() {
    sessionSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
        navigatorKey: navigatorKey,
        title: 'Tryout TKA',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF3157D5)),
          useMaterial3: true,
          scaffoldBackgroundColor: const Color(0xFFF4F7FB),
          cardTheme: const CardThemeData(margin: EdgeInsets.zero),
        ),
        home: const LoginScreen(),
      );
}
