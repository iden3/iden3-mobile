package com.example.iden3core;
import io.flutter.plugin.common.MethodChannel;
import android.util.Log;

import iden3mobile.Callback;

// CHANNEL (flutter <== android <== go)
public class CallbackHandler implements Callback {
    private MainActivity activity;
    private MethodChannel.Result result;

    public CallbackHandler(MainActivity _activity, MethodChannel.Result _result) {
      activity = _activity;
      result = _result;
    }
    
    @Override
    public void verifierResponse(boolean p0, java.lang.Exception p1) {
      callFlutter(p0, p1);
    }

    private void callFlutter(final boolean p0, final java.lang.Exception p1){
      activity.runOnUiThread (new Runnable() {
          public void run() {
            if (p1 == null) {
              result.success(p0);
            } else {
              result.error("proveClaim", "Error proofing claim to issuer: " + p1.getMessage(), null);
            }
          }
      });
    }
}