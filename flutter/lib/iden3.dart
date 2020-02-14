import 'dart:async';

import 'package:flutter/services.dart';  
  

const platform = const MethodChannel('iden3');

// NEW ID
Future<void> newIdentity(String pass, alias, Function(MethodCall) eventHandler) async {
  var arguments = Map();
  arguments["pass"] = pass;
  arguments["alias"] = alias;
  try {
    await platform.invokeMethod("newIdentity", arguments);
    MethodChannel('iden3/events/' + alias).setMethodCallHandler(eventHandler);
  } on PlatformException catch (e) {
    throw(e);
  }
}
// REQUEST CLAIM
Future<Map> requestClaim(String url) async {
  Map ticket;
  var arguments = Map();
  arguments["url"] = url;
  try {
    ticket = await platform.invokeMethod("requestClaim", arguments);
  } on PlatformException catch (e) {
    throw(e);
  }

  if (ticket != null) {
    return ticket;
  }
  throw("Unsuccessful request");
}
// LIST CLAIMS
Future<List<dynamic>> listClaims() async {
  var claims;
  try {
    claims = await platform.invokeMethod("listClaims", Map());
  } on PlatformException catch (e) {
    throw(e);
  }
  return claims;
}
// PPROOF CLAIM
Future<bool> proveClaim(String url, int i) async {
  var arguments = Map();
  arguments["url"] = url;
  arguments["claimIndex"] = i;
  try {
    print("lets proof");
    return await platform.invokeMethod("proveClaim", arguments);
  } on PlatformException catch (e) {
    throw(e);
  }
}
// LIST TICKETS
Future<List<dynamic>> listTickets() async {
  var tickets;
  try {
    tickets = await platform.invokeMethod("listTickets", Map());
  } on PlatformException catch (e) {
    throw(e);
  }
  return tickets;
}