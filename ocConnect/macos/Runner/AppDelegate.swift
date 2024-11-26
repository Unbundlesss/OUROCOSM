import Cocoa
import FlutterMacOS

@main
class AppDelegate: FlutterAppDelegate {
  override func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    return true
  }
  
  override func applicationDidBecomeActive(_ notification: Notification) {
      signal(SIGPIPE, SIG_IGN);//Ignore signal
  }
      
  override func applicationWillUnhide(_ notification: Notification) {
      signal(SIGPIPE, SIG_IGN);
  }
  
  override func applicationWillBecomeActive(_ notification: Notification) {
      signal(SIGPIPE, SIG_IGN);
  }  
}
