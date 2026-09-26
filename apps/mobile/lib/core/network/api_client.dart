import 'dart:async';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

const _accessTokenKey = 'tka_access_token';
const _refreshTokenKey = 'tka_refresh_token';

class ApiFailure implements Exception {
  const ApiFailure(this.message, {this.statusCode, this.networkError = false});
  final String message;
  final int? statusCode;
  final bool networkError;
  @override
  String toString() => message;
}

class TokenStore {
  TokenStore({FlutterSecureStorage? storage})
    : _storage = storage ?? const FlutterSecureStorage();
  final FlutterSecureStorage _storage;
  Future<String?> readAccess() => _storage.read(key: _accessTokenKey);
  Future<String?> readRefresh() => _storage.read(key: _refreshTokenKey);
  Future<void> writeAccess(String token) =>
      _storage.write(key: _accessTokenKey, value: token);
  Future<void> writeRefresh(String token) =>
      _storage.write(key: _refreshTokenKey, value: token);
  Future<void> clear() async {
    await _storage.delete(key: _accessTokenKey);
    await _storage.delete(key: _refreshTokenKey);
  }
}

class ApiClient {
  ApiClient({Dio? dio, TokenStore? tokenStore})
    : _dio =
          dio ??
          Dio(
            BaseOptions(
              baseUrl: const String.fromEnvironment(
                'API_BASE_URL',
                defaultValue: 'http://10.0.2.2:8080/api/v1',
              ),
              connectTimeout: const Duration(seconds: 8),
              receiveTimeout: const Duration(seconds: 10),
              sendTimeout: const Duration(seconds: 8),
              headers: {
                'Accept': 'application/json',
                'Content-Type': 'application/json',
              },
            ),
          ),
      _tokens = tokenStore ?? TokenStore() {
    _dio.interceptors.add(
      QueuedInterceptorsWrapper(
        onRequest: (options, handler) async {
          final token = await _tokens.readAccess();
          if (token != null && token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          handler.next(options);
        },
        onError: (error, handler) async {
          final path = error.requestOptions.path;
          final isCredentialRequest =
              path.endsWith('/auth/login') ||
              path.endsWith('/auth/register') ||
              path.endsWith('/auth/refresh');
          final alreadyRetried =
              error.requestOptions.extra['authRetried'] == true;
          if (error.response?.statusCode == 401 &&
              !isCredentialRequest &&
              !alreadyRetried) {
            final refreshed = await _refreshAccessToken();
            if (refreshed) {
              final token = await _tokens.readAccess();
              error.requestOptions.headers['Authorization'] = 'Bearer $token';
              error.requestOptions.extra['authRetried'] = true;
              final retryClient = Dio();
              try {
                handler.resolve(await retryClient.fetch(error.requestOptions));
                return;
              } on DioException {
                // The common expiry path below clears the invalid session.
              } finally {
                retryClient.close(force: true);
              }
            }
            await _expireSession(error.response?.data);
          } else if (error.response?.statusCode == 401 && alreadyRetried) {
            await _expireSession(error.response?.data);
          }
          handler.next(error);
        },
      ),
    );
  }

  Future<void> _expireSession(dynamic data) async {
    await _tokens.clear();
    final message = data is Map
        ? (data['message'] ?? data['error'] ?? 'Sesi berakhir.').toString()
        : 'Sesi berakhir.';
    if (!_sessionExpired.isClosed) _sessionExpired.add(message);
  }

  final Dio _dio;
  final TokenStore _tokens;
  final StreamController<String> _sessionExpired =
      StreamController<String>.broadcast();
  Stream<String> get sessionExpired => _sessionExpired.stream;

  Future<Map<String, dynamic>> get(String path) async =>
      _request(() => _dio.get<Map<String, dynamic>>(path));
  Future<Map<String, dynamic>> post(
    String path, [
    Map<String, dynamic>? body,
    Map<String, dynamic>? headers,
  ]) async => _request(
    () => _dio.post<Map<String, dynamic>>(
      path,
      data: body ?? const <String, dynamic>{},
      options: Options(headers: headers),
    ),
  );

  Future<void> login(String email, String password) async {
    final response = await _requestResponse(
      () => _dio.post<Map<String, dynamic>>(
        '/auth/login',
        data: {'email': email, 'password': password},
      ),
    );
    final token = response.data?['access_token'];
    if (token is! String || token.isEmpty) {
      throw const ApiFailure('Respons login tidak valid.');
    }
    final refreshToken = _refreshCookie(response);
    if (refreshToken == null) {
      throw const ApiFailure('Server tidak memberikan sesi yang valid.');
    }
    await _tokens.writeAccess(token);
    await _tokens.writeRefresh(refreshToken);
  }

  Future<bool> _refreshAccessToken() async {
    final refreshToken = await _tokens.readRefresh();
    if (refreshToken == null || refreshToken.isEmpty) return false;
    final refreshClient = Dio(
      BaseOptions(
        baseUrl: _dio.options.baseUrl,
        connectTimeout: _dio.options.connectTimeout,
        receiveTimeout: _dio.options.receiveTimeout,
        sendTimeout: _dio.options.sendTimeout,
        headers: const {'Accept': 'application/json'},
      ),
    );
    try {
      final response = await refreshClient.post<Map<String, dynamic>>(
        '/auth/refresh',
        data: const <String, dynamic>{},
        options: Options(headers: {'Cookie': 'tka_refresh=$refreshToken'}),
      );
      final accessToken = response.data?['access_token'];
      final rotatedRefreshToken = _refreshCookie(response);
      if (accessToken is! String ||
          accessToken.isEmpty ||
          rotatedRefreshToken == null) {
        return false;
      }
      await _tokens.writeAccess(accessToken);
      await _tokens.writeRefresh(rotatedRefreshToken);
      return true;
    } on DioException {
      return false;
    } finally {
      refreshClient.close(force: true);
    }
  }

  String? _refreshCookie(Response<dynamic> response) {
    for (final header
        in response.headers.map['set-cookie'] ?? const <String>[]) {
      final match = RegExp(r'(?:^|;\s*)tka_refresh=([^;]+)').firstMatch(header);
      if (match != null && match.group(1)!.isNotEmpty) return match.group(1);
    }
    return null;
  }

  Future<Map<String, dynamic>> _request(
    Future<Response<Map<String, dynamic>>> Function() request,
  ) async {
    final response = await _requestResponse(request);
    return response.data ?? <String, dynamic>{};
  }

  Future<Response<Map<String, dynamic>>> _requestResponse(
    Future<Response<Map<String, dynamic>>> Function() request,
  ) async {
    try {
      return await request();
    } on DioException catch (error) {
      final data = error.response?.data;
      final message = data is Map
          ? (data['message'] ?? data['error'])?.toString()
          : null;
      final offline =
          error.type == DioExceptionType.connectionError ||
          error.type == DioExceptionType.connectionTimeout ||
          error.type == DioExceptionType.unknown;
      throw ApiFailure(
        message ??
            (offline
                ? 'Tidak dapat terhubung ke server.'
                : 'Permintaan gagal.'),
        statusCode: error.response?.statusCode,
        networkError: offline,
      );
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
