class TryoutPackage {
  const TryoutPackage(
      {required this.id,
      required this.title,
      required this.description,
      required this.price,
      required this.validityDays});
  final String id;
  final String title;
  final String description;
  final double price;
  final int validityDays;

  factory TryoutPackage.fromJson(Map<String, dynamic> json) => TryoutPackage(
        id: json['id'] as String,
        title: json['title'] as String,
        description: json['description'] as String,
        price: (json['price'] as num).toDouble(),
        validityDays: (json['validity_days'] as num).toInt(),
      );
}

class CheckoutResult {
  const CheckoutResult({required this.paymentUrl, required this.invoiceNumber});
  final String paymentUrl;
  final String invoiceNumber;

  factory CheckoutResult.fromJson(Map<String, dynamic> json) => CheckoutResult(
        paymentUrl: json['payment_url'] as String,
        invoiceNumber: (json['transaction']
            as Map<String, dynamic>)['invoice_number'] as String,
      );
}
