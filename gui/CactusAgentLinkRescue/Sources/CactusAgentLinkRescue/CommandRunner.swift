import Foundation

final class CommandRunner: @unchecked Sendable {
    private let outputLimit = 1_000_000

    func run(executable: URL, args: [String], timeout: TimeInterval) async -> CommandResult {
        await withCheckedContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                continuation.resume(returning: self.runSync(executable: executable, args: args, timeout: timeout))
            }
        }
    }

    func runSync(executable: URL, args: [String], timeout: TimeInterval) -> CommandResult {
        let started = Date()
        let process = Process()
        let stdoutPipe = Pipe()
        let stderrPipe = Pipe()
        process.executableURL = executable
        process.arguments = args
        process.standardOutput = stdoutPipe
        process.standardError = stderrPipe

        let stdoutBuffer = OutputBuffer(limit: outputLimit)
        let stderrBuffer = OutputBuffer(limit: outputLimit)
        var timedOut = false
        var exitCode: Int32 = -1
        var stderrPrefix = ""

        do {
            stdoutPipe.fileHandleForReading.readabilityHandler = { handle in
                stdoutBuffer.append(handle.availableData)
            }
            stderrPipe.fileHandleForReading.readabilityHandler = { handle in
                stderrBuffer.append(handle.availableData)
            }
            try process.run()
            let deadline = Date().addingTimeInterval(timeout)
            while process.isRunning && Date() < deadline {
                Thread.sleep(forTimeInterval: 0.05)
            }
            if process.isRunning {
                timedOut = true
                process.terminate()
            }
            process.waitUntilExit()
            exitCode = process.terminationStatus
        } catch {
            stderrPrefix = error.localizedDescription
        }
        stdoutPipe.fileHandleForReading.readabilityHandler = nil
        stderrPipe.fileHandleForReading.readabilityHandler = nil
        stdoutBuffer.append(stdoutPipe.fileHandleForReading.availableData)
        stderrBuffer.append(stderrPipe.fileHandleForReading.availableData)

        let ended = Date()
        let stdout = redact(stdoutBuffer.string())
        let stderrBody = redact(stderrBuffer.string())
        let stderr = [stderrPrefix, stderrBody].filter { !$0.isEmpty }.joined(separator: "\n")
        return CommandResult(
            exitCode: exitCode,
            stdout: stdout,
            stderr: stderr,
            duration: ended.timeIntervalSince(started),
            commandDisplay: commandDisplay(executable: executable, args: args),
            startedAt: started,
            endedAt: ended,
            timedOut: timedOut
        )
    }

    private func commandDisplay(executable: URL, args: [String]) -> String {
        ([shellQuote(executable.path)] + args.map(shellQuote)).joined(separator: " ")
    }
}

final class OutputBuffer: @unchecked Sendable {
    private let limit: Int
    private var data = Data()
    private var truncated = false
    private let lock = NSLock()

    init(limit: Int) {
        self.limit = limit
    }

    func append(_ chunk: Data) {
        guard !chunk.isEmpty else { return }
        lock.lock()
        defer { lock.unlock() }
        if data.count >= limit {
            truncated = true
            return
        }
        let remaining = limit - data.count
        if chunk.count > remaining {
            data.append(chunk.prefix(remaining))
            truncated = true
        } else {
            data.append(chunk)
        }
    }

    func string() -> String {
        lock.lock()
        let copy = data
        let wasTruncated = truncated
        lock.unlock()
        var text = String(data: copy, encoding: .utf8) ?? ""
        if wasTruncated {
            text += "\n[output truncated]"
        }
        return text
    }
}

func shellQuote(_ value: String) -> String {
    if value.rangeOfCharacter(from: CharacterSet(charactersIn: " \t\n\"'\\$`")) == nil {
        return value
    }
    return "'" + value.replacingOccurrences(of: "'", with: "'\\''") + "'"
}

func redact(_ input: String) -> String {
    var text = input
    let patterns = [
        #"(?i)(Authorization:\s*Bearer\s+)[A-Za-z0-9._\-+/=]+"#,
        #"(?i)((?:token|access_token|password|passwd)=)[^&\s]+"#,
        #"(?i)\b(?:sk|sk-ant|sk-or|deepseek)[-_][A-Za-z0-9_\-]{8,2048}"#,
        #"(?i)(https?://)[^:\s/@]+:[^@\s/]+@"#,
        #"(?i)((?:HTTP_PROXY|HTTPS_PROXY|ALL_PROXY|http_proxy|https_proxy|all_proxy)=)(\S+)"#
    ]
    for pattern in patterns {
        text = text.replacingOccurrences(of: pattern, with: "$1REDACTED", options: .regularExpression)
    }
    return text
}
