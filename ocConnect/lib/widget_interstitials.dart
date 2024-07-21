// ocConnect | OUROCOSM client tool | GPLv3 | ishani.org 2024
// -----------------------------------------------------------------------------

import 'package:flutter/material.dart';

class LoadingSpinnerPage extends StatelessWidget {
  const LoadingSpinnerPage({super.key});
  @override
  Widget build(BuildContext context) {
    return const Scaffold(
        body: Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          CircularProgressIndicator(),
          SizedBox(height: 10), // Add spacing between spinner and text
          Text("Loading Configuration...")
        ],
      ),
    ));
  }
}

class BootErrorPage extends StatelessWidget {
  const BootErrorPage({super.key, required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
        body: Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Text("Error During Startup",
              style: TextStyle(fontWeight: FontWeight.bold)),
          Text(message)
        ],
      ),
    ));
  }
}
