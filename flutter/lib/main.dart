import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

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
  int _counter = 0;
  static const platform = const MethodChannel('iden3.com/callinggo');
  static const serverIP = "192.168.68.126";

  void initState() {
    super.initState();
    // CHANNEL (flutter <== android)
    platform.setMethodCallHandler(goAsyncHandler);
  }
  // ASYNC TEST
  Future<void> _iden3Action() async {
    String ticket;
    try {
      var arguments = Map();
      arguments["url"] = "http://"+serverIP+":1234/issueClaim";
      ticket = await platform.invokeMethod("requestClaim", arguments);
    } on PlatformException catch (e) {
      print("PlatformException: ${e.message}");
    }

    if (ticket != null) {
      setState(() {
        _message = "waiting issuer: " + ticket;
      });
    }
  }

  Future<void> _incr() async {
    setState(() {
        ++_counter;
    });
  }

  Future<void> goAsyncHandler(MethodCall methodCall) async {
    print("someone just called goAsyncHandler");
    print(methodCall);
    print("_________________________________________________________");
    switch (methodCall.method) {
      case 'onIssuerResponse':
        setState(() {
          _message = methodCall.arguments;
        });
        return;
      default:
        print("UNIMPLEMENTED");
    }
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
              alignment: Alignment.bottomRight,
              child: FloatingActionButton(
                onPressed: _iden3Action,
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
