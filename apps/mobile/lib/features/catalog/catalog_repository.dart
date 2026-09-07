import 'dart:convert';
import 'dart:math';

import '../../core/network/api_client.dart';
import 'catalog_models.dart';

class CatalogRepository {
  const CatalogRepository(this._api);
  final ApiClient _api;

  Future<List<TryoutPackage>> listPackages() async {
    final response = await _api.get('/packages');
    return (response['data'] as List<dynamic>)
        .map((item) => TryoutPackage.fromJson(item as Map<String, dynamic>))
        .toList(growable: false);
  }

  Future<CheckoutResult> checkout(String packageId, String method) async {
    final random = Random.secure();
    final key =
        base64Url.encode(List<int>.generate(24, (_) => random.nextInt(256)));
    final response = await _api.post(
      '/transactions/checkout',
      {'package_id': packageId, 'payment_method': method},
      {'Idempotency-Key': key},
    );
    return CheckoutResult.fromJson(response);
  }
}
