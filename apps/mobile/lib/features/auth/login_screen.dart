import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/network/api_client.dart';
import '../catalog/catalog_screen.dart';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});
  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final email = TextEditingController(text: 'siswa@example.com');
  final password = TextEditingController(text: 'demo123');
  bool loading = false;
  String? error;
  Future<void> login() async {
    setState(() {
      loading = true;
      error = null;
    });
    try {
      await ref.read(apiClientProvider).login(email.text, password.text);
      if (mounted) {
        Navigator.pushReplacement(
            context, MaterialPageRoute(builder: (_) => const CatalogScreen()));
      }
    } catch (e) {
      if (mounted) {
        setState(() => error = e.toString().replaceFirst('Exception: ', ''));
      }
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
      body: SafeArea(
          child: Center(
              child: SingleChildScrollView(
                  padding: const EdgeInsets.all(24),
                  child: Card(
                      child: Padding(
                          padding: const EdgeInsets.all(24),
                          child: Column(
                              mainAxisSize: MainAxisSize.min,
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                const CircleAvatar(
                                    radius: 26, child: Text('T')),
                                const SizedBox(height: 20),
                                Text('Selamat datang',
                                    style: Theme.of(context)
                                        .textTheme
                                        .headlineSmall,
                                    textAlign: TextAlign.center),
                                const SizedBox(height: 20),
                                TextField(
                                    controller: email,
                                    decoration: const InputDecoration(
                                        labelText: 'Email',
                                        border: OutlineInputBorder())),
                                const SizedBox(height: 12),
                                TextField(
                                    controller: password,
                                    obscureText: true,
                                    decoration: const InputDecoration(
                                        labelText: 'Password',
                                        border: OutlineInputBorder())),
                                if (error != null)
                                  Padding(
                                      padding: const EdgeInsets.only(top: 12),
                                      child: Text(error!,
                                          style: const TextStyle(
                                              color: Colors.red))),
                                const SizedBox(height: 20),
                                FilledButton(
                                    onPressed: loading ? null : login,
                                    child: Text(
                                        loading ? 'Memproses...' : 'Masuk'))
                              ])))))));
}
