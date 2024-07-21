// ocConnect | OUROCOSM client tool | GPLv3 | ishani.org 2024
// -----------------------------------------------------------------------------

import 'dart:async';
import 'dart:math';

import 'package:bitsdojo_window/bitsdojo_window.dart';
import 'package:flutter/material.dart';
import 'dart:collection';
import 'server_config.dart';
import 'widget_server_actions.dart';
import 'widget_interstitials.dart';
import 'ndls_icons.dart' if (dart.library.io) 'ndls_icons.dart';

// -----------------------------------------------------------------------------
Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  runApp(const OcConnect());

  // configure default window display via bitsdojo
  doWhenWindowReady(() {
    const initialSize = Size(800, 500);
    appWindow.minSize = initialSize;
    appWindow.maxSize = const Size(800, 1200);
    appWindow.size = initialSize;
    appWindow.alignment = Alignment.center;
    appWindow.title = "OUROCOSM Connect";
    appWindow.show();
  });
}

// -----------------------------------------------------------------------------
// OcConnect initial state; shows a spinner while we finish loading configs and doing
// network resolution, then diverts to the FrontPage to show server choice
class _OcConnectState extends State<OcConnect> {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
        title: 'OUROCOSM Connect',
        theme: ThemeData(
            fontFamily: "FiraCode",
            colorScheme: ColorScheme.fromSeed(seedColor: Colors.tealAccent),
            useMaterial3: true,
            scrollbarTheme: ScrollbarThemeData(
              thumbVisibility: WidgetStateProperty.all<bool>(true),
            )),
        home: Scaffold(
            body: FutureBuilder<List<OuroServerConfig>>(
          future: loadServerConfigs(),
          builder: (context, snapshot) {
            switch (snapshot.connectionState) {
              // spin while loading configs task completes
              case ConnectionState.waiting:
                return const LoadingSpinnerPage();
              default:
                // failure to parse the config data
                if (snapshot.hasError) {
                  return BootErrorPage(message: '${snapshot.error}');
                }
                // empty config data; in theory this would have failed parsing already
                if (snapshot.data?.isEmpty ?? true) {
                  return const BootErrorPage(
                      message: 'No server configurations provided');
                }
                // we have configs to work with, continue on
                return FrontPage(
                    configs: snapshot.data!, title: 'Select Private Server');
            }
          },
        )));
  }
}

class OcConnect extends StatefulWidget {
  const OcConnect({super.key});
  @override
  State<OcConnect> createState() => _OcConnectState();
}

// -----------------------------------------------------------------------------
// the front page shows the set of servers loaded from the config data
class FrontPage extends StatefulWidget {
  const FrontPage({super.key, required this.title, required this.configs});

  final String title;
  final List<OuroServerConfig> configs;

  @override
  State<FrontPage> createState() => _FrontPageState();
}

// -----------------------------------------------------------------------------
// front page state includes instances of server reachability trackers that
// poke the targets every so often to see if they are still available
// and any other data we get from that process (ie. most recent public riff)
class _FrontPageState extends State<FrontPage> {
  // reachability tracker per server
  late final List<OuroServerTracker> serverTrackers = [];
  // subscriptions made to the trackers' event stream, updates to reachability
  // are captured and acted upon by these
  late final List<StreamSubscription<OuroServerStateUpdate>>
      serverSubscriptions = [];
  HashMap<String, OuroServerStateUpdate> serverStateData = HashMap();

  @override
  void initState() {
    super.initState();
    var rng = Random();

    // create a reachability tracker per known server
    for (final serverConfig in widget.configs) {
      serverTrackers.add(OuroServerTracker(config: serverConfig));

      // bind to the event stream for reachability updates; on arrival,
      // update our state, putting the data into a map of server-name:server-state
      final subscription = serverTrackers.last.eventBroadcast
          .listen((OuroServerStateUpdate data) {
        setState(() {
          serverStateData[serverConfig.displayName] = data;
        });
      });
      serverSubscriptions.add(subscription);

      // kick off tracking once every so often
      var secondsBetweenUpdate = rng.nextInt(20) + 30;
      serverTrackers.last.start(Duration(seconds: secondsBetweenUpdate));
    }
  }

  // stop and toss all the trackers and subscriptions
  @override
  void dispose() {
    for (final serverSub in serverSubscriptions) {
      serverSub.cancel();
    }
    for (final serverTracker in serverTrackers) {
      serverTracker.stop();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final localColourScheme = Theme.of(context).colorScheme;
    return Scaffold(
        appBar: AppBar(
          backgroundColor: localColourScheme.inversePrimary,
          title: Text(widget.title,
              style: const TextStyle(
                  fontFamily: 'D-DIN',
                  fontWeight: FontWeight.w400,
                  fontSize: 32.0)),
        ),
        // list of all servers
        body: ListView.separated(
            padding: const EdgeInsets.all(24),
            itemCount: serverTrackers.length,
            separatorBuilder: (context, index) => const SizedBox(
                  height: 20,
                ),
            itemBuilder: (BuildContext context, int index) {
              var serverState =
                  serverStateData[serverTrackers[index].config.displayName];

              IconData displayIcon = Icons.question_mark_rounded;
              String displayStatus = "Status Unknown";
              Color displayColor =
                  localColourScheme.inverseSurface.withAlpha(100);

              // change colours/icons based on reachability state
              if (serverState != null) {
                if (serverState.isReachable) {
                  displayIcon = Ndls.endlessslogo;
                  displayStatus =
                      "Last public riff posted ${serverState.serverLastRiffDelta}";
                  displayColor = localColourScheme.inverseSurface;
                } else {
                  displayIcon = Icons.warning_amber_rounded;
                  displayStatus = serverState.reachability;
                  displayColor = localColourScheme.error;
                }
              }

              return FilledButton.icon(
                style: ButtonStyle(
                    backgroundColor: WidgetStateProperty.all(displayColor),
                    alignment: Alignment.centerLeft,
                    padding: WidgetStateProperty.all<EdgeInsets>(
                        const EdgeInsets.all(24))),
                icon: Icon(
                  displayIcon,
                  size: 64.0,
                ),
                onPressed: () {
                  // switch to the server page, we have some kind of state
                  // even if the state is "unreachable"
                  if (serverState != null) {
                    Navigator.push(
                      context,
                      MaterialPageRoute(
                          builder: (context) => ServerActionsPage(
                              serverLastState: serverState,
                              serverStream:
                                  serverTrackers[index].eventBroadcast,
                              config: serverTrackers[index].config)),
                    );
                  }
                },
                label: Padding(
                  padding: const EdgeInsets.only(left: 12.0),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(serverTrackers[index].config.displayName,
                          style: const TextStyle(
                              fontFamily: 'D-DIN',
                              fontWeight: FontWeight.w600,
                              fontSize: 18.0)),
                      Text(serverTrackers[index].config.displayBio),
                      Text(displayStatus),
                    ],
                  ),
                ),
              );
            }) // This trailing comma makes auto-formatting nicer for build methods.
        );
  }
}
