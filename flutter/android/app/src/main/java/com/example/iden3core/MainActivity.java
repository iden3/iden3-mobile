package com.example.iden3core;

import androidx.annotation.NonNull;
import io.flutter.embedding.android.FlutterActivity;
import io.flutter.embedding.engine.FlutterEngine;
import io.flutter.plugin.common.MethodChannel;
import io.flutter.plugins.GeneratedPluginRegistrant;

import iden3mobile.Identity;

public class MainActivity extends FlutterActivity {
  // Comunication with Flutter
  MethodChannel channel;
  CallbackHandler callback;

  // Iden3 declaration
  Identity iden3;

  @Override
  public void configureFlutterEngine(@NonNull FlutterEngine flutterEngine) {
    // INITALIZE IDEN3 AND CHANNELS (flutter <==> android <==> go)
    // CHANNEL (android ==> go)
    iden3 = new Identity();
    // String storagePath = getFilesDir().getAbsolutePath();
    // iden3.setPath(storagePath);
    // CHANNEL (flutter ==> android)
    channel = new MethodChannel(flutterEngine.getDartExecutor().getBinaryMessenger(), "iden3.com/callinggo");
    // CHANNEL (flutter <== android <== go)
    callback = new CallbackHandler(this, channel);
    iden3.setCallbackHandler(callback);

    GeneratedPluginRegistrant.registerWith(flutterEngine);
    // HANDLE METHODS (flutter ==> android)
    channel.setMethodCallHandler((call, result) -> {
      // NEW ID
      if (call.method.equals("newID")) {
        try {
            iden3.createIdentity();
            result.success(true);
            return;
        } catch (Exception e) {
            result.error("newID", e.getMessage(), null);
        }
      }
      // ISSUE CLAIM
      if (call.method.equals("requestClaim")) {
        if (!call.hasArgument("url")) {
            result.error("requestClaim", "Send argument as Map<\"data\", string>", null);
            return;
        }
        try {
            String url = call.argument("url");
            String ticket = iden3.requestClaim(url);
            result.success(ticket);
            return;
        } catch (Exception e) {
            result.error("requestClaim", e.getMessage(), null);
        }
        return;
      }
      // UNKNOWN METHOD
      else {
        result.notImplemented();
      }
    });
  }
}


