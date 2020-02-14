package com.example.iden3core;
import java.util.*;
import android.util.Log;
import androidx.annotation.NonNull;
import io.flutter.embedding.android.FlutterActivity;
import io.flutter.embedding.engine.FlutterEngine;
import io.flutter.plugin.common.MethodChannel;
import io.flutter.plugins.GeneratedPluginRegistrant;

import iden3mobile.Identity;
import iden3mobile.BytesArray;
import iden3mobile.Ticket;
import iden3mobile.TicketsMap;
import iden3mobile.TicketsMapInterface;

public class MainActivity extends FlutterActivity {
  // Comunication with Flutter
  MethodChannel channel;
  EventHandler event;

  // Iden3 declaration
  Identity iden3;

  @Override
  public void configureFlutterEngine(@NonNull FlutterEngine flutterEngine) {
    // INITALIZE IDEN3 AND CHANNELS (flutter <==> android <==> go)
    // CHANNEL (android ==> go)
    String storePath = getFilesDir().getAbsolutePath();
    Log.println(Log.ERROR, "MainActivity", "Storage path: " + storePath);
    // CHANNEL (flutter ==> android)
    channel = new MethodChannel(flutterEngine.getDartExecutor().getBinaryMessenger(), "iden3");
    GeneratedPluginRegistrant.registerWith(flutterEngine);
    // HANDLE METHODS (flutter ==> android)
    channel.setMethodCallHandler((call, result) -> {
      Log.println(Log.ERROR, "CALL:", "Method " + call.method);

      // CREATE ID
      if (call.method.equals("newIdentity")) {
        // TODO: Expose interface to add extra genesis claims
        if (!call.hasArgument("alias") || !call.hasArgument("pass")) {
          result.error("newIdentity", "Send argument as Map. Mandatory arguments: alias (string), pass (string).", null);
          return;
        }
        String alias = call.argument("alias");
        try {
            iden3 = new Identity(storePath + "/" + alias, call.argument("pass"), new BytesArray(), new EventHandler(this, flutterEngine, alias));
            // CHANNEL (flutter <== android <== go)
            result.success(true);
            return;
        } catch (Exception e) {
            result.error("newIdentity", e.getMessage(), null);
            return;
        }
      }
      // LOAD ID
      else if (call.method.equals("loadIdentity")) {
        if (!call.hasArgument("alias")) {
          result.error("loadIdentity", "Send argument as Map<\"alias\", string>", null);
          return;
        }
        String alias = call.argument("alias");
        try {
            iden3 = new Identity(storePath + "/" + alias, new EventHandler(this, flutterEngine, alias));
            result.success(true);
            return;
        } catch (Exception e) {
            result.error("newIdentity", e.getMessage(), null);
            return;
        }
      }
      // REQUEST CLAIM
      else if (call.method.equals("requestClaim")) {
        if (!call.hasArgument("url")) {
            result.error("requestClaim", "Send argument as Map<\"url\", string>", null);
            return;
        }
        try {
            String data = (call.hasArgument("data"))? call.argument("data") : "";
            Ticket ticket = iden3.requestClaim(call.argument("url"), data);
            result.success(ticketToMap(ticket));
            return;
        } catch (Exception e) {
            result.error("requestClaim", e.getMessage(), null);
            return;
        }
      }
      // LIST CLAIMS
      else if (call.method.equals("listClaims")) {
        long nClaims = iden3.getReceivedClaimsLen();
        List<byte[]> claims = new ArrayList<byte[]>((int) nClaims);
        Log.println(Log.ERROR, "listClaims", "Parsing " + String.valueOf(nClaims) + " claims");
        for (long i = 0; i < nClaims; ++i) {
          try {
            claims.add((int) i, iden3.getReceivedClaim(i));
          } catch (Exception e) {
            result.error("listClaims", e.getMessage(), null);
            return;
          }
        }
        result.success(claims);
        return;
      }
      // PROOF CLAIM
      else if (call.method.equals("proveClaim")) {
        if (!call.hasArgument("url") || !call.hasArgument("claimIndex")) {
            result.error("proveClaim", "Send argument as Map. Mandatory arguments: url (string), claimIndex (int).", null);
            return;
        }
        long claimIndex = new Long((int) call.argument("claimIndex"));
        iden3.proveClaim(call.argument("url"), claimIndex, new CallbackHandler(this, result));
        return; //CallbackHandler will call result
      }
      // LIST TICKETS
      else if (call.method.equals("listTickets")) {
        TicketsMap tickets = iden3.getTickets();
        List<HashMap> parsedTickets = new ArrayList<HashMap>();
        try {
          tickets.forEach(new TicketsMapInterface() {
            @Override
            public void f(Ticket ticket) throws Exception {
              parsedTickets.add(ticketToMap(ticket));
            }
          });
          result.success(parsedTickets);
          return;
        } catch (Exception e) {
          result.error("listTickets", e.getMessage(), null);
        }
      }
      // UNKNOWN METHOD
      else {
        result.notImplemented();
        return;
      }
    });
  }

  private HashMap ticketToMap(Ticket t) {
    HashMap tMap = new HashMap();
    tMap.put("id", t.getId());
    tMap.put("lastChecked", t.getLastChecked());
    tMap.put("type", t.getType());
    return tMap;
  }
}


