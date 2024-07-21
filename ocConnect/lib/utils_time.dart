// ocConnect | OUROCOSM client tool | GPLv3 | ishani.org 2024
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
int unixTimeDeltaSecondsFromNow(int timestamp) {
  final now = DateTime.now();
  final difference =
      now.difference(DateTime.fromMillisecondsSinceEpoch(timestamp));
  final seconds = difference.inSeconds.abs();
  return seconds;
}

// -----------------------------------------------------------------------------
String unixTimeDeltaToHumanReadable(int timestamp) {
  final seconds = unixTimeDeltaSecondsFromNow(timestamp);
  if (seconds < 60) {
    return 'just now';
  } else if (seconds < 3600) {
    final minutes = seconds ~/ 60;
    return '$minutes minute${minutes > 1 ? 's' : ''} ago';
  } else if (seconds < 86400) {
    final hours = seconds ~/ 3600;
    return '$hours hour${hours > 1 ? 's' : ''} ago';
  } else {
    final days = seconds ~/ 86400;
    return '$days day${days > 1 ? 's' : ''} ago';
  }
}
