// ocConnect | OUROCOSM client tool | GPLv3 | ishani.org 2024
// -----------------------------------------------------------------------------

import 'package:flutter/material.dart';
import 'package:flutter/services.dart' as services;
import 'package:http/http.dart' as http;
import 'package:http/http.dart';
import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:yaml/yaml.dart';
import 'utils_time.dart';

// -----------------------------------------------------------------------------
// describes a server we could connect to, given the chance. these are written into
// the server list YAML - or in the future received from a master list server -
// loaded at boot time and serve as the core data for listing out and connecting to
// a given private server
class OuroServerConfig {
  final String displayName;
  final String displayBio;
  final String displayGeo;
  final String scheme;
  final String host;
  final int apiPort;
  final int dbPort;

  late String?
      hostIPv4; // resolved ip4, or null if we didn't manage to resolve it

  OuroServerConfig({
    required this.displayName,
    required this.displayBio,
    required this.displayGeo,
    required this.scheme,
    required this.host,
    required this.apiPort,
    required this.dbPort,
  });

  // loads a server target representation from a key/value map, usually from YAML
  factory OuroServerConfig.fromKV(dynamic keyValueRoot) {
    final keyValueMap = keyValueRoot as Map<dynamic, dynamic>?;
    if (keyValueMap == null) {
      throw const FormatException(
          "Server configuration value invalid type, expects map of key/values");
    }

    final missingKeys = [];

    // check on the presence of the required keys
    var keysToCheck = [
      'display-name',
      'display-bio',
      'display-geo',
      'scheme',
      'host',
      'api-port',
      'db-port',
    ];
    for (var key in keysToCheck) {
      if (keyValueMap[key] == null) {
        missingKeys.add(key);
      }
    }

    if (missingKeys.isNotEmpty) {
      throw FormatException(
          "Required values missing from configuration: ${missingKeys.join(', ')}");
    }

    return OuroServerConfig(
      displayName: keyValueMap['display-name'] as String,
      displayBio: keyValueMap['display-bio'] as String,
      displayGeo: keyValueMap['display-geo'] as String,
      scheme: keyValueMap['scheme'] as String,
      host: keyValueMap['host'] as String,
      apiPort: keyValueMap['api-port'] as int,
      dbPort: keyValueMap['db-port'] as int,
    );
  }
}

// -----------------------------------------------------------------------------
// API /cosm/v1/status
// data returned to indicate the current state of the server
class CosmStatus {
  final int version; // arbitrary version
  final bool awake; // if false, server is under maintenance
  final int serverTime; // the current unix time on the server
  final int
      mostRecentPublicJamChange; // the unix time of the last committed riff in the publics

  CosmStatus(this.version, this.awake, this.serverTime,
      this.mostRecentPublicJamChange);

  CosmStatus.fromJson(Map<String, dynamic> json)
      : version = json['version'] as int,
        awake = json['awake'] as bool,
        serverTime = json['serverTime'] as int,
        mostRecentPublicJamChange = json['mostRecentPublicJamChange'] as int;

  Map<String, dynamic> toJson() => {
        'version': version,
        'awake': awake,
        'serverTime': serverTime,
        'mostRecentPublicJamChange': mostRecentPublicJamChange,
      };
}

// -----------------------------------------------------------------------------
// the immutable block of data we pass around representing a completed
// reachability pass on a given server; these are generated every N seconds/minutes
// and flow out to the UI to show the state of server targets
class OuroServerStateUpdate {
  const OuroServerStateUpdate(
      {required this.updateTimestamp,
      required this.reachability,
      required this.serverLastRiffDelta,
      required this.serverTimeDelta,
      required this.isReachable});

  // when was this update made, local unix timestamp
  final int updateTimestamp;
  // broad reachability string, eg. "Server is reachable" or "Nah it's knackered mate"
  final String reachability;
  // human-readable rep of how long ago a public riff was committed, eg. "4 minutes ago"
  final String serverLastRiffDelta;
  // delta in seconds between local unix time and server unix time; if big, that could be a problem for Endlesss
  final int serverTimeDelta;
  // is this server reachable at all?
  final bool isReachable;
}

// -----------------------------------------------------------------------------
// an object used to manage regular poking of a server to see what state its in;
// is it reachable, is anyone jamming, things of that nature
class OuroServerTracker {
  OuroServerTracker({required this.config}) {
    eventBroadcast = _updateController.stream.asBroadcastStream();
  }

  // we use the async Stream system to create a subscribe'able channel of
  // update events that UI things can listen to
  final _updateController = StreamController<OuroServerStateUpdate>();
  late Stream<OuroServerStateUpdate> eventBroadcast;

  // the server we will be tracking
  OuroServerConfig config;
  Timer? _reachabilityTimer;
  bool? _isReachable;

  // the callback function kicked by our periodic timer to do the actual work
  // of calling the server
  void _runReachability(Timer t) async {
    // abort if we didn't manage to fetch an IP address, cancel the timer
    // this may be harsh; we could try periodically re-acquiring the IP?
    if (config.hostIPv4 == null) {
      debugPrint(
          '${config.displayName} no server IP resolved, aborting tracking');

      _updateController.sink.add(OuroServerStateUpdate(
          updateTimestamp: DateTime.now().millisecondsSinceEpoch,
          reachability: "Count not resolve IP address for `${config.host}`",
          serverLastRiffDelta: "",
          serverTimeDelta: 0,
          isReachable: false));

      _isReachable = false;
      t.cancel();
      return;
    }

    // ping the status endpoint, see what happens
    // if anything throws, catch and mark the server as problematic
    String failureStatus = "";
    try {
      // 4 seconds might be too stingy a timeout!
      http.Response cosmStatus = await http
          .get(Uri.parse(
              '${config.scheme}://${config.hostIPv4}:${config.apiPort}/cosm/v1/status'))
          .timeout(const Duration(seconds: 4));

      // try and decode the result to our expected structure
      CosmStatus cosmStatusResult = CosmStatus.fromJson(
          jsonDecode(cosmStatus.body) as Map<String, dynamic>);

      // all good so far?
      if (cosmStatus.statusCode == 200) {
        // push out a successful state update block
        _updateController.sink.add(OuroServerStateUpdate(
            updateTimestamp: DateTime.now().millisecondsSinceEpoch,
            reachability: "Server is reachable",
            serverLastRiffDelta: unixTimeDeltaToHumanReadable(
                cosmStatusResult.mostRecentPublicJamChange),
            serverTimeDelta:
                unixTimeDeltaSecondsFromNow(cosmStatusResult.serverTime),
            isReachable: true));

        // flip the reachable flag, log if it just changed
        if (_isReachable == null || _isReachable == false) {
          _isReachable = true;
          debugPrint('${config.displayName} server became reachable');
        }
        return; // early out, this is a successful reachability check
      }
      // drop out to the problem zone, catching any context we can on the way
    } on TimeoutException catch (e) {
      debugPrint('Timeout talking to server; ${e.message}');
      failureStatus = "; connection unresponsive";
    } on ClientException catch (e) {
      debugPrint('Failed to contact server; ${e.message}');
    } on FormatException catch (e) {
      debugPrint('Failed to unmarshal json to CosmStatus; ${e.message}');
      failureStatus = "; invalid status format";
    } catch (e) {
      debugPrint('Unknown error checking server state; ${e.toString()}');
    }

    // server no bueno
    _updateController.sink.add(OuroServerStateUpdate(
        updateTimestamp: DateTime.now().millisecondsSinceEpoch,
        reachability: "Failed to contact server$failureStatus",
        serverLastRiffDelta: "",
        serverTimeDelta: 0,
        isReachable: false));

    if (_isReachable == null || _isReachable == true) {
      _isReachable = false;
      debugPrint('${config.displayName} server became unreachable');
    }
  }

  // begin checking reachability!
  void start(Duration durationBetweenChecks) {
    debugPrint('${config.displayName} starting reachability checks');
    _reachabilityTimer =
        Timer.periodic(durationBetweenChecks, _runReachability);
    _runReachability(_reachabilityTimer!);
  }

  // stop checking reachability!
  void stop() {
    debugPrint('${config.displayName} stopping reachability checks');
    _reachabilityTimer?.cancel();
  }
}

// async load of our YAML config, will throw exceptions on any errors
Future<List<OuroServerConfig>> loadServerConfigs() async {
  const String configYamlFilename = "ourocosm.client.yaml";

  List<OuroServerConfig> result = [];

  String configContent =
      await services.rootBundle.loadString('assets/$configYamlFilename');

  // load the yaml and check we have a root list of servers as expected
  final YamlMap yamlRoot = loadYaml(configContent);
  final serverList = yamlRoot['servers'] as List<dynamic>?;
  if (serverList == null) {
    throw const FormatException(
        "'servers' root key not found in '$configYamlFilename', or is the incorrect type");
  }

  // copy out each server declaration into our list
  for (var i = 0; i < serverList.length; i++) {
    final serverConfig = OuroServerConfig.fromKV(serverList[i]);

    serverConfig.hostIPv4 = null;
    try {
      // lookup the IP address for the server
      var ipv4Addresses = await InternetAddress.lookup(serverConfig.host,
          type: InternetAddressType.IPv4);

      if (ipv4Addresses.isNotEmpty) {
        serverConfig.hostIPv4 = ipv4Addresses[0].address;
      }
    } on Exception catch (e) {
      debugPrint(
          "Failed to resolve hostname `${serverConfig.host}`, ${e.toString()}");
    }

    result.add(serverConfig);
  }

  // sometimes it's nice to watch a spinner spin!
  await Future.delayed(const Duration(milliseconds: 500));

  return result;
}
