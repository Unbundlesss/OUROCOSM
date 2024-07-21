// ocConnect | OUROCOSM client tool | GPLv3 | ishani.org 2024
// -----------------------------------------------------------------------------

import 'package:flutter/material.dart';
import 'dart:io';
import 'server_config.dart';

// -----------------------------------------------------------------------------
// the launch page turns up to show we're about to try and run an app, a
// brief distraction while the gears spin
class LaunchHostPage extends StatefulWidget {
  const LaunchHostPage(
      {super.key, required this.launchExecutable, required this.config});

  final OuroServerConfig config; // where to connect to
  final String launchExecutable; // what to run

  @override
  State<LaunchHostPage> createState() => _LaunchHostPage();
}

// -----------------------------------------------------------------------------
class _LaunchHostPage extends State<LaunchHostPage> {
  Future<Process> launch() async {
    // the magic env vars that rewires Studio to get it talking to a new server
    final envVars = <String, String>{
      'ENDLESSS_ENV': 'local',
      'ENDLESSS_DATA_URL': '${widget.config.hostIPv4}:${widget.config.dbPort}',
      'ENDLESSS_API_URL': '${widget.config.hostIPv4}:${widget.config.apiPort}',
      'ENDLESSS_WEB_URL': '${widget.config.hostIPv4}:${widget.config.apiPort}',
      'ENDLESSS_HTTPS': "false",
    };

    var processInstance = Process.start(widget.launchExecutable, [],
        environment: envVars, mode: ProcessStartMode.detached);

    await Future.delayed(const Duration(seconds: 1));

    return processInstance;
  }

  Widget launchInProgress() {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text("Launching", style: TextStyle(fontWeight: FontWeight.bold)),
        const SizedBox(height: 10),
        Text(widget.launchExecutable),
      ],
    );
  }

  Widget launchComplete() {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text("Launch Complete",
            style: TextStyle(fontWeight: FontWeight.bold)),
        const SizedBox(height: 10),
        IconButton(
            onPressed: () => {Navigator.pop(context)},
            icon: const Icon(Icons.arrow_back)),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
        body: Center(
            child: FutureBuilder<Process>(
                future: launch(),
                builder: (context, snapshot) {
                  switch (snapshot.connectionState) {
                    // spin while loading configs task completes
                    case ConnectionState.waiting:
                      return launchInProgress();
                    default:
                      return launchComplete();
                  }
                })));
  }
}
