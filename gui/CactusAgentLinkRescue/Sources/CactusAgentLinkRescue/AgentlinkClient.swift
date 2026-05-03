import Foundation

final class AgentlinkClient {
    let packageRoot: URL
    let binaryURL: URL
    private let runner = CommandRunner()

    init(packageRoot: URL? = nil) {
        let root = packageRoot ?? AgentlinkClient.locatePackageRoot()
        self.packageRoot = root
        self.binaryURL = root.appendingPathComponent("bin/agentlink")
    }

    static func locatePackageRoot() -> URL {
        if let override = ProcessInfo.processInfo.environment["AGENTLINK_GUI_AGENTLINK_ROOT"], !override.isEmpty {
            return URL(fileURLWithPath: override)
        }
        if let resourceURL = Bundle.main.resourceURL {
            let bundled = resourceURL.appendingPathComponent("agentlink")
            if FileManager.default.fileExists(atPath: bundled.appendingPathComponent("bin/agentlink").path) {
                return bundled
            }
        }
        let cwd = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        let candidates = [
            cwd.appendingPathComponent("dist/Cactus-AgentLink-Rescue"),
            cwd,
            cwd.deletingLastPathComponent()
        ]
        for candidate in candidates where FileManager.default.fileExists(atPath: candidate.appendingPathComponent("bin/agentlink").path) {
            return candidate
        }
        return cwd.appendingPathComponent("Contents/Resources/agentlink")
    }

    func run(_ args: [String], timeout: TimeInterval = 30) async -> CommandResult {
        await runner.run(executable: binaryURL, args: args, timeout: timeout)
    }

    func runJSON<T: Decodable>(_ type: T.Type, args: [String], timeout: TimeInterval = 30) async -> (CommandResult, T?) {
        let result = await run(args, timeout: timeout)
        let decoded = try? JSONDecoder().decode(T.self, from: Data(result.stdout.utf8))
        return (result, decoded)
    }

    func copyableTerminalCommand(_ args: [String], sudo: Bool = false) -> String {
        let cd = "cd " + shellQuote(packageRoot.path)
        let command = (sudo ? ["sudo", "./bin/agentlink"] : ["./bin/agentlink"]) + args
        return cd + "\n" + command.map(shellQuote).joined(separator: " ")
    }

    func fallbackRescueCommand(level: String) -> String {
        "cd \(shellQuote(packageRoot.path))\nsudo /bin/bash ./rescue.sh \(level)"
    }

    func fieldMode() -> FieldMode? {
        let url = packageRoot.appendingPathComponent("field-mode.json")
        guard let data = try? Data(contentsOf: url) else { return nil }
        return try? JSONDecoder().decode(FieldMode.self, from: data)
    }
}

enum GUISelftest {
    static func run() -> Int32 {
        let client = AgentlinkClient()
        let runner = CommandRunner()
        let start = Date()
        let version = runner.runSync(executable: client.binaryURL, args: ["version"], timeout: 10)
        let doctor = runner.runSync(executable: client.binaryURL, args: ["doctor", "--json"], timeout: 30)
        let brain = runner.runSync(executable: client.binaryURL, args: ["brain", "doctor", "--json"], timeout: 30)
        let guided = runner.runSync(executable: client.binaryURL, args: ["guided", "rescue", "--target", "auto", "--dry-run", "--json"], timeout: 120)
        let installer = runner.runSync(executable: client.binaryURL, args: ["installer", "doctor", "--json"], timeout: 30)
        let readiness = runner.runSync(executable: client.binaryURL, args: ["readiness", "doctor", "--json"], timeout: 90)
        let summary: [String: Any] = [
            "ok": version.succeeded && doctor.succeeded && brain.succeeded && guided.succeeded && installer.succeeded && readiness.succeeded,
            "packageRoot": client.packageRoot.path,
            "binaryPath": client.binaryURL.path,
            "versionExitCode": version.exitCode,
            "doctorExitCode": doctor.exitCode,
            "brainDoctorExitCode": brain.exitCode,
            "guidedExitCode": guided.exitCode,
            "installerDoctorExitCode": installer.exitCode,
            "readinessExitCode": readiness.exitCode,
            "durationMs": Int(Date().timeIntervalSince(start) * 1000),
            "version": version.stdout.trimmingCharacters(in: .whitespacesAndNewlines),
            "brainDoctor": jsonObject(brain.stdout) ?? brain.stdout,
            "guidedRescue": jsonObject(guided.stdout) ?? guided.stdout,
            "installerDoctor": jsonObject(installer.stdout) ?? installer.stdout,
            "readiness": jsonObject(readiness.stdout) ?? readiness.stdout
        ]
        let data = try? JSONSerialization.data(withJSONObject: summary, options: [.prettyPrinted, .sortedKeys])
        if let data, let text = String(data: data, encoding: .utf8) {
            print(redact(text))
        }
        return (version.succeeded && doctor.succeeded && brain.succeeded && guided.succeeded && installer.succeeded && readiness.succeeded) ? 0 : 1
    }

    private static func jsonObject(_ text: String) -> Any? {
        guard let data = text.data(using: .utf8) else { return nil }
        return try? JSONSerialization.jsonObject(with: data)
    }

	static func runLongOutput() -> Int32 {
		let runner = CommandRunner()
		let generator = URL(fileURLWithPath: FileManager.default.fileExists(atPath: "/usr/bin/python3") ? "/usr/bin/python3" : "/usr/bin/perl")
		let marker = "GUI_LONG_OUTPUT_MARKER"
		let args: [String]
		if generator.path.hasSuffix("python3") {
			args = ["-c", "import sys; sys.stdout.write('sk-test-GUI_LONG_OUTPUT_MARKER-' + 'A'*1200000)"]
		} else {
			args = ["-e", "print 'sk-test-GUI_LONG_OUTPUT_MARKER-' . ('A' x 1200000)"]
		}
		let long = runner.runSync(executable: generator, args: args, timeout: 20)
		let timeout = runner.runSync(executable: URL(fileURLWithPath: "/bin/sleep"), args: ["2"], timeout: 0.2)
		let ok = long.succeeded &&
			long.stdout.contains("[output truncated]") &&
			!long.stdout.contains(marker) &&
			timeout.timedOut
        let summary: [String: Any] = [
            "ok": ok,
			"longOutputExitCode": long.exitCode,
			"longOutputLength": long.stdout.count,
			"longOutputTruncated": long.stdout.contains("[output truncated]"),
			"redactionClean": !long.stdout.contains(marker),
			"timeoutTimedOut": timeout.timedOut,
			"timeoutExitCode": timeout.exitCode
		]
        let data = try? JSONSerialization.data(withJSONObject: summary, options: [.prettyPrinted, .sortedKeys])
        if let data, let text = String(data: data, encoding: .utf8) {
            print(redact(text))
        }
        return ok ? 0 : 1
    }
}
