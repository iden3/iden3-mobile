import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'iden3.dart' as iden3;

void main() => runApp(MyApp());

class MyApp extends StatelessWidget {
  // This widget is the root of your application.
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Iden3',
      theme: ThemeData(
        primarySwatch: Colors.green,
      ),
      home: MyHomePage(title: 'Iden3'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  MyHomePage({Key key, this.title}) : super(key: key);

  final String title;

  @override
  _MyHomePageState createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  // set callback handler
  String _message = "press button";
  String _endpoint = "http://192.168.200.181:1234";
  int _counter = 0;

  void initState() {
    super.initState();
    // CHANNEL (flutter <== android)
    iden3.newIdentity("mySecurePassword:)", "myID", _eventHandler).then(
      (onValue) => print("Identity created")
    ).catchError((e) => print(e));
  }

  // EVENT HANDLER
  Future<void> _eventHandler(MethodCall methodCall) async {
    switch (methodCall.method) {
      case 'onIssuerResponse':
        print("EVENT onIssuerResponse");
        if(methodCall.arguments == "new claim"){
          iden3.listClaims().then((claims) {
            if (claims.length == 0) {
              print("CLAIM NOT LISTED, DEMO HAS FAILED");
            }
            iden3.proveClaim(_endpoint + "/verifyClaim", 0).then((success) {
              if(success){
                print("DEMO COMPLETED :)");
              }
              else {
                print("CLAIM NOT VERIFIED, DEMO HAS FAILED");
              }
            }).catchError((e) => print(e));
          }).catchError((e) {
            print("ERROR LISTING CLAIMS");
            print(e);
          });
        } else {
          print("ERROR REQUESTING CLAIM: " + methodCall.arguments);
        }
        return; 
      default:
        print("UNIMPLEMENTED");
    }
  }

  Future<void> _doDemo() async {
    Map ticket = await iden3.requestClaim(_endpoint + "/issueClaim").catchError((e) => print(e));
    print("CLAIM REQUEST TICKET: ");
    _printTicket(ticket);
  }

  void _printTicket(Map ticket){
    ticket.forEach((k, v) => print('--- ${k}: ${v}'));
  }

  Future<void> _listTickets() async {
    var tickets =  await iden3.listTickets().catchError((e) => print(e));
    if (tickets.length == 0){
      print("NO PENDING TICKETS");
      return;
    }
    print("PENDING TICKETS:");
    for (var t in tickets) {
      _printTicket(t);
      print("--------------");
    }
  }

  Future<void> _incr() async {
    setState(() {
        ++_counter;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title),
      ),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: <Widget>[
            Text(
              'You have pushed the button this many times:',
            ),
            Text(
              _message,
              style: Theme.of(context).textTheme.display1,
            ),
            Text(
              "$_counter",
              style: Theme.of(context).textTheme.display1,
            ),
          ],
        ),
      ),
      floatingActionButton: Stack(
          children: <Widget>[
            Align(
              alignment: Alignment.bottomLeft,
              child: FloatingActionButton(
                onPressed: _incr,
                tooltip: 'Increment',
                child: Icon(Icons.add),
              ),
            ),
            Align(
              alignment: Alignment.bottomCenter,
              child: FloatingActionButton(
                onPressed: _listTickets,
                tooltip: 'Tickets',
                child: Icon(Icons.album),
              ),
            ),
            Align(
              alignment: Alignment.bottomRight,
              child: FloatingActionButton(
                onPressed: _doDemo,
                tooltip: 'Wait',
                child: Icon(Icons.alarm),
              ),
            ),
          ],
        ),
       // This trailing comma makes auto-formatting nicer for build methods.
    );
  }
}
