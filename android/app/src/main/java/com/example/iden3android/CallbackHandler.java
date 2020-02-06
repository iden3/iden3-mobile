package com.example.iden3android;

import android.util.Log;
import android.widget.TextView;

import org.w3c.dom.Text;

import iden3mobile.Callback;

// CHANNEL (android <== go)
public class CallbackHandler implements Callback {
    private MainActivity activity;
    private TextView tv;

    public CallbackHandler(MainActivity _activity, TextView _tv){
        tv = _tv;
        activity = _activity;
    }

    @Override
    public void onIssuerResponse(String ticket, String id, String claim, java.lang.Exception error) {
        setText("\nReceived response for the ticket: "+ticket+". Claim: "+claim);
        Log.println(Log.ERROR, "CB:onIssuerResponse", "ticket: "+ticket+"\nid: "+id+"\nclaim: "+claim+"\nerror: "+error);
    }


    @Override
    public void onVerifierResponse(String ticket, String id, boolean aproved, java.lang.Exception error) {
        Log.println(Log.ERROR, "CB:onIssuerResponse", "ticket: "+ticket+"\nid: "+id+"\naproved: "+aproved+"\nerror: "+error);
    }

    private void setText(final String txt){
        activity.runOnUiThread (new Runnable() {
            public void run() {
                tv.setText(txt);
            }
        });
    }
}
