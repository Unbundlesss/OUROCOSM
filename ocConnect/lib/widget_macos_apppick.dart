import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'dart:async';
import 'dart:io' show Platform;

// -----------------------------------------------------------------------------
// represents a discovered app on MacOS, a display name and full path to the
// actual executable file to launch
class ApplicationInfo {
  late final String displayName;
  late final String executablePath;

  ApplicationInfo(Map<String, String> data) {
    displayName = data["displayName"] as String;
    executablePath = data["executablePath"] as String;
  }
}

// -----------------------------------------------------------------------------
// event sent back to indicate selection event
class AppSelectionRequest extends Notification {
  final ApplicationInfo appInfo;
  AppSelectionRequest(this.appInfo);
}

// -----------------------------------------------------------------------------
// interface to custom platform-specific code in MainFlutterWindow.swift
class PlatformSpecific {
  static const MethodChannel _channel = MethodChannel('platform_specific');

  // painfully convert the mystical unknown object pushed back over the extension channel
  // into a list of <string, string> values
  static List<Map<String, String>> convertList(List<Object?> list) {
    return list.whereType<Map>().cast<Map<Object?, Object?>>().map((map) {
      try {
        return map.cast<String, String>();
      } catch (e) {
        debugPrint('Error converting map: $e');
        return <String, String>{};
      }
    }).toList();
  }

  // fetch the list of applications and bundle data from /Applications
  // via the custom extension code, turn it into ApplicationInfo objects
  static Future<List<ApplicationInfo>> getApplications() async {
    final result = await _channel.invokeMethod('getApplications');
    if (result == null) return [];

    final List<Map<String, String>> validData = convertList(result);

    List<ApplicationInfo> appInfoResult = [];
    for (var i = 0; i < validData.length; i++) {
      appInfoResult.add(ApplicationInfo(validData[i]));
    }
    return appInfoResult;
  }
}

// -----------------------------------------------------------------------------
class MacOSApplicationPickerPage extends StatefulWidget {
  const MacOSApplicationPickerPage(
      {super.key,
      required this.applicationInfoStream,
      required this.applicationsAlreadyAdded});
  final StreamController<ApplicationInfo> applicationInfoStream;
  final Set<String> applicationsAlreadyAdded;

  @override
  MacOSApplicationPickerPageState createState() =>
      MacOSApplicationPickerPageState();
}

// -----------------------------------------------------------------------------
class MacOSApplicationPickerPageState
    extends State<MacOSApplicationPickerPage> {
  List<ApplicationInfo> _knownApps = [];

  @override
  void initState() {
    super.initState();
    if (Platform.isMacOS) {
      _findMacOSApps();
    }
  }

  void _findMacOSApps() async {
    final appsList = await PlatformSpecific.getApplications();
    setState(() {
      _knownApps = appsList;
    });
  }

  void _handleSelection(BuildContext context, ApplicationInfo appInfo) {
    AppSelectionRequest(appInfo).dispatch(context);
    widget.applicationInfoStream.sink.add(appInfo);
    Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        title: const Text("MacOS Applications",
            style: TextStyle(
                fontFamily: 'D-DIN',
                fontWeight: FontWeight.w600,
                fontSize: 30.0)),
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView.builder(
              itemCount: _knownApps.length,
              itemBuilder: (context, index) {
                final appInfo = _knownApps[index];
                final appAlreadyAdded = widget.applicationsAlreadyAdded
                    .contains(appInfo.executablePath);
                final disabledGrey = Colors.black.withAlpha(64);

                if (appAlreadyAdded) {
                  return AbsorbPointer(absorbing:true, child: ListTile(
                      leading: IconButton(
                          icon: const Icon(Icons.check_circle_outline),
                          color: disabledGrey,
                          onPressed: () => {}),
                      title: Text(appInfo.displayName,
                          style: TextStyle(
                              fontWeight: FontWeight.bold,
                              color: disabledGrey))));
                }
                return ListTile(
                  leading: IconButton(
                    icon: const Icon(Icons.add),
                    onPressed: () => {_handleSelection(context, appInfo)},
                  ),
                  title: Text(appInfo.displayName,
                      style: const TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: Text(appInfo.executablePath),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
