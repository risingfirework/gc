import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _tokenKey = 'tka_access_token';

class ApiFailure implements Exception {
  const ApiFailure(this.message, {this.statusCode, this.networkError = false});
  final String message;
  final int? statusCode;
  final bool networkError;
  @override
  String toString() => message;
}

class TokenStore {
  Future<String?> read() async =>
      (await SharedPreferences.getInstance()).getString(_tokenKey);
  Future<void> write(String token) async =>
      (await SharedPreferences.getInstance()).setString(_tokenKey, token);
  Future<void> clear() async =>
      (await SharedPreferences.getInstance()).remove(_tokenKey);
}

class ApiClient {
  ApiClient({Dio? dio, TokenStore? tokenStore})
      : _dio = dio ??
            Dio(BaseOptions(
              baseUrl: const String.fromEnvironment('API_BASE_URL',
                  defaultValue: 'http://10.0.2.2:8080/api/v1'),
              connectTimeout: const Duration(seconds: 8),
              receiveTimeout: const Duration(seconds: 10),
              sendTimeout: const Duration(seconds: 8),
              headers: {
                'Accept': 'application/json',
                'Content-Type': 'application/json'
              },
            )),
        _tokens = tokenStore ?? TokenStore() {
    _dio.interceptors.add(QueuedInterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await _tokens.read();
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        final path = error.requestOptions.path;
        final isCredentialRequest =
            path.endsWith('/auth/login') || path.endsWith('/auth/register');
        if (error.response?.statusCode == 401 && !isCredentialRequest) {
          await _tokens.clear();
          final data = error.response?.data;
          final message = data is Map
              ? (data['message'] ?? data['error'] ?? 'Sesi berakhir.')
                  .toString()
              : 'Sesi berakhir.';
          if (!_sessionExpired.isClosed) _sessionExpired.add(message);
        }
        handler.next(error);
      },
    ));
  }

  final Dio _dio;
  final TokenStore _tokens;
  final StreamController<String> _sessionExpired =
      StreamController<String>.broadcast();
  Stream<String> get sessionExpired => _sessionExpired.stream;

  Future<Map<String, dynamic>> get(String path) async =>
      _request(() => _dio.get<Map<String, dynamic>>(path));
  Future<Map<String, dynamic>> post(String path,
          [Map<String, dynamic>? body, Map<String, dynamic>? headers]) async =>
      _request(() => _dio.post<Map<String, dynamic>>(path,
          data: body ?? const <String, dynamic>{},
          options: Options(headers: headers)));

  Future<void> login(String email, String password) async {
    final response =
        await post('/auth/login', {'email': email, 'password': password});
    final token = response['access_token'];
    if (token is! String || token.isEmpty) {
      throw const ApiFailure('Respons login tidak valid.');
    }
    await _tokens.write(token);
  }

  Future<Map<String, dynamic>> _request(
      Future<Response<Map<String, dynamic>>> Function() request) async {
    try {
      final response = await request();
      return response.data ?? <String, dynamic>{};
    } on DioException catch (error) {
      final data = error.response?.data;
      final message =
          data is Map ? (data['message'] ?? data['error'])?.toString() : null;
      final offline = error.type == DioExceptionType.connectionError ||
          error.type == DioExceptionType.connectionTimeout ||
          error.type == DioExceptionType.unknown;
      throw ApiFailure(
          message ??
              (offline
                  ? 'Tidak dapat terhubung ke server.'
                  : 'Permintaan gagal.'),
          statusCode: error.response?.statusCode,
          networkError: offline);
    }
  }

  void dispose() {
    _sessionExpired.close();
    _dio.close(force: true);
  }
}

final apiClientProvider = Provider<ApiClient>((ref) {
  final client = ApiClient();
  ref.onDispose(client.dispose);
  return client;
});
