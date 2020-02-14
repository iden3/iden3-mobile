package com.example.iden3core;
import io.flutter.embedding.engine.FlutterEngine;
import io.flutter.plugin.common.MethodChannel;
import android.util.Log;

import iden3mobile.Event;

// CHANNEL (flutter <== android <== go)
public class EventHandler implements Event {
    private MainActivity activity;
    MethodChannel channel;

    public EventHandler(MainActivity _activity, FlutterEngine flutterEngine, String alias) {
      activity = _activity;
      channel = new MethodChannel(flutterEngine.getDartExecutor().getBinaryMessenger(), "iden3/events/" + alias);
    }
    
    @Override
    public void onIssuerResponse(String ticket, String id, byte[] claim, java.lang.Exception error) {
        Log.println(Log.ERROR, "CB:onIssuerResponse", "ticket: "+ticket+"\nid: "+id+"\nclaim: "+claim+"\nerror: "+error);
        if(error != null){
          callFlutter("onIssuerResponse", "claim not received due an error: " + error.getMessage());
          return;
        }
        callFlutter("onIssuerResponse", "new claim");
    }

    private void callFlutter(final String function, final String arguments){
      activity.runOnUiThread (new Runnable() {
          public void run() {
            channel.invokeMethod(function, arguments);
          }
      });
    }
}