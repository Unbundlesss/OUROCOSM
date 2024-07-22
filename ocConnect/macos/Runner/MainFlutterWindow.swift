import Cocoa
import FlutterMacOS
import bitsdojo_window_macos
import Foundation

struct ApplicationInfo {
    let displayName: String
    let executablePath: String
}

// check if a URL points to an application directory (has an Info.plist)
extension URL {
    var isApplicationDirectory: Bool {
        let infoPlistURL = appendingPathComponent("Contents/Info.plist")
        return FileManager.default.fileExists(atPath: infoPlistURL.path)
    }
}

func getApplicationsInfo() -> [ApplicationInfo] {
    let applicationsURL = URL(fileURLWithPath: "/Applications")
    
    var applicationInfoList = [ApplicationInfo]()
    
    do {
        let applicationURLs = try FileManager.default.contentsOfDirectory(at: applicationsURL, includingPropertiesForKeys: nil, options: [])
        for applicationURL in applicationURLs {
            guard applicationURL.isApplicationDirectory else { continue }
            
            let infoPlistURL = applicationURL.appendingPathComponent("Contents/Info.plist")
            guard let infoDictionary = try? NSDictionary(contentsOf: infoPlistURL) else { continue }
            
            guard let executablePath = infoDictionary["CFBundleExecutable"] as? String else { continue }
            let displayName = infoDictionary["CFBundleDisplayName"] as? String ?? executablePath
            
            let appInfo = ApplicationInfo(displayName: displayName, executablePath: applicationURL.appendingPathComponent("Contents/MacOS/\(executablePath)").path)
            applicationInfoList.append(appInfo)
        }
    } catch {
        print("Error fetching application info: \(error)")
    }
    
    return applicationInfoList
}

class MainFlutterWindow: BitsdojoWindow {
  override func bitsdojo_window_configure() -> UInt {
  return 0
  }
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)

    let channel = FlutterMethodChannel(name: "platform_specific", binaryMessenger: flutterViewController.engine.binaryMessenger)
    channel.setMethodCallHandler { (call, result) in
      switch call.method {
      case "getApplications":
        let applicationInfos = getApplicationsInfo()
        let dartData = applicationInfos.map { appInfo in
            [
                "displayName": appInfo.displayName,
                "executablePath": appInfo.executablePath
            ]
        }
        result(dartData)
      default:
        result(FlutterMethodNotImplemented)
      }
    }    
    
    RegisterGeneratedPlugins(registry: flutterViewController)

    super.awakeFromNib()
  }
}
