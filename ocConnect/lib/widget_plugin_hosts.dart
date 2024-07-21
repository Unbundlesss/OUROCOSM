// ocConnect | OUROCOSM client tool | GPLv3 | ishani.org 2024
// -----------------------------------------------------------------------------

import 'dart:convert';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:win32_registry/win32_registry.dart';
import 'package:path/path.dart' as path;
import 'dart:io' show Platform;
import 'dart:io' as io;

// -----------------------------------------------------------------------------
// event sent to trigger an app launch
class HostLaunchRequest extends Notification {
  final String executable;
  HostLaunchRequest(this.executable);
}

// -----------------------------------------------------------------------------
// holds the serialised definition of a plugin host to launch
class PluginHost {
  final String name;
  final String path;

  PluginHost(this.name, this.path);

  Map<String, dynamic> toJson() => {'name': name, 'path': path};

  // Factory constructor to create PluginHost from JSON
  factory PluginHost.fromJson(Map<String, dynamic> json) =>
      PluginHost(json['name'] as String, json['path'] as String);
}

// -----------------------------------------------------------------------------
// the plugin list handles all apps we can launch, including Studio
// (if we can find it automatically)
class PluginHostList extends StatefulWidget {
  const PluginHostList({super.key});

  @override
  _PluginHostListState createState() => _PluginHostListState();
}

class _PluginHostListState extends State<PluginHostList> {
  List<PluginHost> _pluginHosts = [];
  String? _builtInStudioHost;

  // Fetch data from SharedPreferences on initialization
  @override
  void initState() {
    super.initState();
    _loadPluginHosts();
    if (Platform.isWindows) {
      _findStudioOnWindowsViaRegistry();
    }
  }

  // get the Studio install path from the registry on Windows
  void _findStudioOnWindowsViaRegistry() {
    const keyPath = r'SOFTWARE\Endlesss Ltd\Endlesss';
    final key = Registry.openPath(RegistryHive.localMachine, path: keyPath);

    var studioInstallPath = key.getValueAsString('InstallationPath');
    if (studioInstallPath != null) {
      debugPrint("Found Studio install reg key: $studioInstallPath");

      studioInstallPath = path.join(studioInstallPath, "Endlesss.exe");
      if (io.File(studioInstallPath).existsSync()) {
        debugPrint("Studio confirmed to exist");
        _builtInStudioHost = studioInstallPath;
      }
    }
    key.close();
  }

  // Load PluginHosts from SharedPreferences
  Future<void> _loadPluginHosts() async {
    final prefs = await SharedPreferences.getInstance();
    final encodedData = prefs.getStringList('oc_plugin_hosts');
    if (encodedData != null) {
      setState(() {
        _pluginHosts = encodedData
            .map((json) => PluginHost.fromJson(jsonDecode(json)))
            .toList();
      });
    }
  }

  // Save PluginHosts to SharedPreferences
  Future<void> _savePluginHosts() async {
    final prefs = await SharedPreferences.getInstance();
    final encodedData =
        _pluginHosts.map((host) => jsonEncode(host.toJson())).toList();
    prefs.setStringList('oc_plugin_hosts', encodedData);
  }

  // Add a new PluginHost instance
  // on Windows, we use a file picker so the user can find an .exe
  void _addPluginHost() async {
    FilePickerResult? result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['exe'],
    );

    if (result != null) {
      setState(() {
        PlatformFile file = result.files.first;
        _pluginHosts
            .add(PluginHost(file.name.replaceAll(".exe", ""), file.path!));
      });
      _savePluginHosts();
    } else {
      // User canceled the picker
    }
  }

  // delete a PluginHost instance
  void _deletePluginHost(int index) {
    setState(() {
      _pluginHosts.removeAt(index);
    });
    _savePluginHosts();
  }

  @override
  Widget build(BuildContext context) {
    final localColourScheme = Theme.of(context).colorScheme;
    return Column(
      children: [
        // if we found Studio automatically, show it as an undeleteable tile first
        if (_builtInStudioHost != null)
          ListTile(
            leading: IconButton(
              icon: const Icon(Icons.play_circle_fill),
              onPressed: () =>
                  {HostLaunchRequest(_builtInStudioHost!).dispatch(context)},
            ),
            title: const Text("Endlesss Studio",
                style: TextStyle(fontWeight: FontWeight.bold)),
            subtitle: Text(_builtInStudioHost!),
          ),
        Expanded(
          child: ListView.builder(
            itemCount: _pluginHosts.length,
            itemBuilder: (context, index) {
              final pluginHost = _pluginHosts[index];
              return ListTile(
                leading: IconButton(
                  icon: const Icon(Icons.play_circle_fill),
                  onPressed: () =>
                      {HostLaunchRequest(pluginHost.path).dispatch(context)},
                ),
                title: Text(pluginHost.name,
                    style: const TextStyle(fontWeight: FontWeight.bold)),
                subtitle: Text(pluginHost.path),
                trailing: IconButton(
                  icon: const Icon(Icons.delete),
                  onPressed: () => _deletePluginHost(index),
                ),
              );
            },
          ),
        ),
        Padding(
          padding: const EdgeInsets.only(top: 16.0),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              ElevatedButton.icon(
                style: ButtonStyle(
                  foregroundColor:
                      WidgetStateProperty.all(localColourScheme.surface),
                  backgroundColor:
                      WidgetStateProperty.all(localColourScheme.inverseSurface),
                ),
                icon: const Icon(Icons.add),
                label: const Text('Add Host ...'),
                onPressed: _addPluginHost,
              )
            ],
          ),
        ),
      ],
    );
  }
}
