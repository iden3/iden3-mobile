package com.example.iden3core;
import io.flutter.plugin.common.MethodChannel;
import android.util.Log;

import iden3mobile.Callback;

// CHANNEL (flutter <== android <== go)
public class CallbackHandler implements Callback {
    private MainActivity activity;
    MethodChannel channel;

    public CallbackHandler(MainActivity _activity, MethodChannel _channel) {
      activity = _activity;
      channel = _channel;
    }
    
    @Override
    public void onIssuerResponse(String ticket, String id, String claim, java.lang.Exception error) {
        Log.println(Log.ERROR, "CB:onIssuerResponse", "ticket: "+ticket+"\nid: "+id+"\nclaim: "+claim+"\nerror: "+error);
        callFlutter("onIssuerResponse", "\nReceived response for the ticket: "+ticket+". Claim: "+claim);
    }


    @Override
    public void onVerifierResponse(String ticket, String id, boolean aproved, java.lang.Exception error) {
        Log.println(Log.ERROR, "CB:onIssuerResponse", "ticket: "+ticket+"\nid: "+id+"\naproved: "+aproved+"\nerror: "+error);
    }

    private void callFlutter(final String function, final String arguments){
      activity.runOnUiThread (new Runnable() {
          public void run() {
            channel.invokeMethod(function, arguments);
          }
      });
    }
}