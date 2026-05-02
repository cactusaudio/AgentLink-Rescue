// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "CactusAgentLinkRescue",
    platforms: [.macOS(.v13)],
    products: [
        .executable(name: "CactusAgentLinkRescue", targets: ["CactusAgentLinkRescue"])
    ],
    targets: [
        .executableTarget(
            name: "CactusAgentLinkRescue",
            path: "Sources/CactusAgentLinkRescue",
            exclude: ["Resources/Info.plist", "Resources/Assets.xcassets"]
        )
    ]
)
