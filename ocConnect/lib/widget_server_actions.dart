// ocConnect | OUROCOSM client tool | GPLv3 | ishani.org 2024
// -----------------------------------------------------------------------------

import 'dart:async';

import 'package:flutter/material.dart';
import 'widget_launch_host.dart';
import 'widget_plugin_hosts.dart';
import 'server_config.dart';

class ServerActionsPage extends StatefulWidget {
  const ServerActionsPage(
      {super.key,
      required this.config,
      required this.serverLastState,
      required this.serverStream});

  final OuroServerConfig config;
  final OuroServerStateUpdate serverLastState;
  final Stream<OuroServerStateUpdate> serverStream;

  @override
  State<ServerActionsPage> createState() => _ServerActionsPage();
}

class ConfigSubtitleWidget extends StatelessWidget
    implements PreferredSizeWidget {
  const ConfigSubtitleWidget(
      {super.key, required this.serverState, required this.config});

  final OuroServerConfig config;
  final OuroServerStateUpdate? serverState;

  @override
  Size get preferredSize => const Size.fromHeight(50.0);

  @override
  Widget build(BuildContext context) {
    final localColourScheme = Theme.of(context).colorScheme;
    Color bar1Color = localColourScheme.primary;
    Color bar2Color = localColourScheme.secondaryFixed;
    Color text1Color = localColourScheme.primaryContainer;
    Color text2Color = localColourScheme.shadow;

    String currentState =
        serverState?.reachability ?? "Awaiting server state ...";
    if (serverState == null) {
      bar2Color = localColourScheme.shadow;
      text2Color = localColourScheme.surfaceContainerHigh;
    } else if (serverState!.isReachable == false) {
      bar2Color = localColourScheme.error;
      text2Color = localColourScheme.errorContainer;
    } else {
      currentState +=
          " (${serverState!.serverTimeDelta} second${serverState!.serverTimeDelta == 1 ? '' : 's'} time delta)";
    }

    return Column(
      children: [
        Container(
          alignment: Alignment.topLeft,
          padding: const EdgeInsets.only(left: 72.0, bottom: 3.0, top: 5.0),
          color: bar1Color,
          child: Text(
            "${config.host} // ${config.hostIPv4}",
            style: TextStyle(color: text1Color, fontSize: 14),
          ),
        ),
        Container(
          alignment: Alignment.topLeft,
          padding: const EdgeInsets.only(left: 72.0, bottom: 3.0, top: 5.0),
          color: bar2Color,
          child: Text(
            currentState,
            style: TextStyle(
                color: text2Color, fontWeight: FontWeight.w600, fontSize: 14),
          ),
        ),
      ],
    );
  }
}

class _ServerActionsPage extends State<ServerActionsPage> {
  StreamSubscription<OuroServerStateUpdate>? streamSubscription;
  OuroServerStateUpdate? lastKnownState;

  @override
  void initState() {
    super.initState();
    lastKnownState = widget.serverLastState;
    streamSubscription =
        widget.serverStream.listen((OuroServerStateUpdate data) {
      setState(() {
        lastKnownState = data;
      });
    });
  }

  @override
  void dispose() {
    streamSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        title: Text(widget.config.displayName,
            style: const TextStyle(
                fontFamily: 'D-DIN',
                fontWeight: FontWeight.w600,
                fontSize: 30.0)),
        bottom: ConfigSubtitleWidget(
            serverState: lastKnownState, config: widget.config),
      ),
      body: Padding(
        padding: const EdgeInsets.all(20.0),
        child: Column(
          children: [
            Expanded(
                child: NotificationListener<HostLaunchRequest>(
                    child: PluginHostList(serverConfig: widget.config),
                    onNotification: (n) {
                      Navigator.push(
                        context,
                        MaterialPageRoute(
                            builder: (context) => LaunchHostPage(
                                launchExecutable: n.executable,
                                config: widget.config)),
                      );
                      return true;
                    }))
          ],
        ),
      ),
    );
  }
}
