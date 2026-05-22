// pubspec.yaml 需要：crypto: ^3.0.0
import 'dart:convert';
import 'package:crypto/crypto.dart';

String signRequest({
  required String clientSecret,
  required String method,
  required String path,
  required int timestamp,
  required String nonce,
  required List<int> bodyBytes,
}) {
  List<int> key;
  try {
    key = base64.decode(clientSecret);
  } catch (_) {
    key = utf8.encode(clientSecret);
  }

  final bodySha = sha256.convert(bodyBytes).toString();
  final payload = '${method.toUpperCase()}\n$path\n$timestamp\n$nonce\n$bodySha';
  final digest = Hmac(sha256, key).convert(utf8.encode(payload));
  return base64.encode(digest.bytes);
}
