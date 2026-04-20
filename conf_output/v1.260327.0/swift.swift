struct AnyCodable {
    let value: Any?

    init(_ value: Any?) {
        self.value = value
    }
}
extension AnyCodable: Codable, @unchecked Sendable {
    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        if container.decodeNil() { value = nil }
        else if let v = try? container.decode(Bool.self) { value = v }
        else if let v = try? container.decode(Int.self) { value = v }
        else if let v = try? container.decode(UInt.self) { value = v }
        else if let v = try? container.decode(Int64.self) { value = v }
        else if let v = try? container.decode(UInt64.self) { value = v }
        else if let v = try? container.decode(Double.self) { value = v }
        else if let v = try? container.decode(String.self) { value = v }
        else if let v = try? container.decode([String: AnyCodable].self) {
            value = v.mapValues { $0.value }
        }
        else if let v = try? container.decode([AnyCodable].self) {
            value = v.map { $0.value }
        }
        else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported JSON value")
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()

        switch value {
        case nil:
            try container.encodeNil()
        case let v as Bool:
            try container.encode(v)
        case let v as Int:
            try container.encode(v)
        case let v as Double:
            try container.encode(v)
        case let v as String:
            try container.encode(v)
        case let v as [String: Any]:
            try container.encode(v.mapValues { AnyCodable($0) })
        case let v as [Any]:
            try container.encode(v.map { AnyCodable($0) })
        default:
            throw EncodingError.invalidValue(value as Any, .init(codingPath: container.codingPath, debugDescription: "Unsupported JSON value"))
        }
    }
}

struct APIConfig {
    let tag: String?
    let listen: String?
    let services: [String]?
}

extension APIConfig: Codable, Sendable {}

struct BalancingRule {
    let tag: String?
    let selector: [String]?
    let strategy: StrategyConfig?
    let fallbackTag: String?
}

extension BalancingRule: Codable, Sendable {}

struct BridgeConfig {
    let tag: String?
    let domain: String?
}

extension BridgeConfig: Codable, Sendable {}

struct BurstObservatoryConfig {
    let subjectSelector: [String]?
    let pingConfig: healthCheckSettings?
}

extension BurstObservatoryConfig: Codable, Sendable {}

struct Config {
    let transport: [String: AnyCodable]?
    let log: LogConfig?
    let routing: RouterConfig?
    let dns: DNSConfig?
    let inbounds: [InboundDetourConfig]?
    let outbounds: [OutboundDetourConfig]?
    let policy: PolicyConfig?
    let api: APIConfig?
    let metrics: MetricsConfig?
    let stats: AnyCodable?
    let reverse: ReverseConfig?
    let fakeDns: AnyCodable?
    let observatory: ObservatoryConfig?
    let burstObservatory: BurstObservatoryConfig?
    let version: VersionConfig?
}

extension Config: Codable, Sendable {}

struct CustomSockoptConfig {
    let system: String?
    let network: String?
    let level: String?
    let opt: String?
    let value: String?
    let type: String?
}

extension CustomSockoptConfig: Codable, Sendable {}

struct DNSConfig {
    let servers: [NameServerConfig?]?
    let hosts: AnyCodable?
    let clientIp: AnyCodable?
    let tag: String?
    let queryStrategy: String?
    let disableCache: Bool?
    let serveStale: Bool?
    let serveExpiredTTL: UInt64?
    let disableFallback: Bool?
    let disableFallbackIfMatch: Bool?
    let enableParallelQuery: Bool?
    let useSystemHosts: Bool?
}

extension DNSConfig: Codable, Sendable {}

struct FinalMask {
    let tcp: [Mask]?
    let udp: [Mask]?
    let quicParams: QuicParamsConfig?
}

extension FinalMask: Codable, Sendable {}

struct GRPCConfig {
    let authority: String?
    let serviceName: String?
    let multiMode: Bool?
    let idle_timeout: Int64?
    let health_check_timeout: Int64?
    let permit_without_stream: Bool?
    let initial_windows_size: Int64?
    let user_agent: String?
}

extension GRPCConfig: Codable, Sendable {
    private enum CodingKeys: String, CodingKey {
        case authority = "authority"
        case serviceName = "serviceName"
        case multiMode = "multiMode"
        case idle_timeout = "idle_timeout"
        case health_check_timeout = "health_check_timeout"
        case permit_without_stream = "permit_without_stream"
        case initial_windows_size = "initial_windows_size"
        case user_agent = "user_agent"
    }
}

struct HappyEyeballsConfig {
    let prioritizeIPv6: Bool?
    let tryDelayMs: UInt64?
    let interleave: UInt64?
    let maxConcurrentTry: UInt64?
}

extension HappyEyeballsConfig: Codable, Sendable {}

struct HttpUpgradeConfig {
    let host: String?
    let path: String?
    let headers: [String: String]?
    let acceptProxyProtocol: Bool?
}

extension HttpUpgradeConfig: Codable, Sendable {}

struct HysteriaConfig {
    let version: Int64?
    let auth: String?
    let congestion: String?
    let up: String?
    let down: String?
    let udphop: UdpHop?
    let udpIdleTimeout: Int64?
    let masquerade: Masquerade?
}

extension HysteriaConfig: Codable, Sendable {}

struct InboundDetourConfig {
    let protocol: String?
    let port: AnyCodable?
    let listen: AnyCodable?
    let settings: AnyCodable?
    let tag: String?
    let streamSettings: StreamConfig?
    let sniffing: SniffingConfig?
}

extension InboundDetourConfig: Codable, Sendable {}

struct KCPConfig {
    let mtu: UInt64?
    let tti: UInt64?
    let uplinkCapacity: UInt64?
    let downlinkCapacity: UInt64?
    let congestion: Bool?
    let readBufferSize: UInt64?
    let writeBufferSize: UInt64?
    let header: AnyCodable?
    let seed: String?
}

extension KCPConfig: Codable, Sendable {}

struct LogConfig {
    let access: String?
    let error: String?
    let loglevel: String?
    let dnsLog: Bool?
    let maskAddress: String?
}

extension LogConfig: Codable, Sendable {}

struct Mask {
    let type: String?
    let settings: AnyCodable?
}

extension Mask: Codable, Sendable {}

struct Masquerade {
    let type: String?
    let dir: String?
    let url: String?
    let rewriteHost: Bool?
    let insecure: Bool?
    let content: String?
    let headers: [String: String]?
    let statusCode: Int64?
}

extension Masquerade: Codable, Sendable {}

struct MetricsConfig {
    let tag: String?
    let listen: String?
}

extension MetricsConfig: Codable, Sendable {}

struct MuxConfig {
    let enabled: Bool?
    let concurrency: Int64?
    let xudpConcurrency: Int64?
    let xudpProxyUDP443: String?
}

extension MuxConfig: Codable, Sendable {}

struct NameServerConfig {
    let address: AnyCodable?
    let clientIp: AnyCodable?
    let port: UInt64?
    let skipFallback: Bool?
    let domains: [String]?
    let expectedIPs: [String]?
    let expectIPs: [String]?
    let queryStrategy: String?
    let tag: String?
    let timeoutMs: UInt64?
    let disableCache: Bool?
    let serveStale: Bool?
    let serveExpiredTTL: UInt64?
    let finalQuery: Bool?
    let unexpectedIPs: [String]?
}

extension NameServerConfig: Codable, Sendable {}

struct ObservatoryConfig {
    let subjectSelector: [String]?
    let probeURL: String?
    let probeInterval: Int64?
    let enableConcurrency: Bool?
}

extension ObservatoryConfig: Codable, Sendable {}

struct OutboundDetourConfig {
    let protocol: String?
    let sendThrough: String?
    let tag: String?
    let settings: AnyCodable?
    let streamSettings: StreamConfig?
    let proxySettings: ProxyConfig?
    let mux: MuxConfig?
    let targetStrategy: String?
}

extension OutboundDetourConfig: Codable, Sendable {}

struct Policy {
    let handshake: UInt64?
    let connIdle: UInt64?
    let uplinkOnly: UInt64?
    let downlinkOnly: UInt64?
    let statsUserUplink: Bool?
    let statsUserDownlink: Bool?
    let statsUserOnline: Bool?
    let bufferSize: Int64?
}

extension Policy: Codable, Sendable {}

struct PolicyConfig {
    let levels: [UInt64: Policy?]?
    let system: SystemPolicy?
}

extension PolicyConfig: Codable, Sendable {}

struct PortalConfig {
    let tag: String?
    let domain: String?
}

extension PortalConfig: Codable, Sendable {}

struct ProxyConfig {
    let tag: String?
    let transportLayer: Bool?
}

extension ProxyConfig: Codable, Sendable {}

struct QuicParamsConfig {
    let congestion: String?
    let debug: Bool?
    let brutalUp: String?
    let brutalDown: String?
    let udpHop: UdpHop?
    let initStreamReceiveWindow: UInt64?
    let maxStreamReceiveWindow: UInt64?
    let initConnectionReceiveWindow: UInt64?
    let maxConnectionReceiveWindow: UInt64?
    let maxIdleTimeout: Int64?
    let keepAlivePeriod: Int64?
    let disablePathMTUDiscovery: Bool?
    let maxIncomingStreams: Int64?
}

extension QuicParamsConfig: Codable, Sendable {}

struct REALITYConfig {
    let masterKeyLog: String?
    let show: Bool?
    let target: AnyCodable?
    let dest: AnyCodable?
    let type: String?
    let xver: UInt64?
    let serverNames: [String]?
    let privateKey: String?
    let minClientVer: String?
    let maxClientVer: String?
    let maxTimeDiff: UInt64?
    let shortIds: [String]?
    let mldsa65Seed: String?
    let limitFallbackUpload: AnyCodable?
    let limitFallbackDownload: AnyCodable?
    let fingerprint: String?
    let serverName: String?
    let password: String?
    let publicKey: String?
    let shortId: String?
    let mldsa65Verify: String?
    let spiderX: String?
}

extension REALITYConfig: Codable, Sendable {}

struct ReverseConfig {
    let bridges: [BridgeConfig]?
    let portals: [PortalConfig]?
}

extension ReverseConfig: Codable, Sendable {}

struct RouterConfig {
    let rules: [AnyCodable]?
    let domainStrategy: String?
    let balancers: [BalancingRule?]?
}

extension RouterConfig: Codable, Sendable {}

struct SniffingConfig {
    let enabled: Bool?
    let destOverride: [String]?
    let domainsExcluded: [String]?
    let metadataOnly: Bool?
    let routeOnly: Bool?
}

extension SniffingConfig: Codable, Sendable {}

struct SocketConfig {
    let mark: Int64?
    let tcpFastOpen: Any?
    let tproxy: String?
    let acceptProxyProtocol: Bool?
    let domainStrategy: String?
    let dialerProxy: String?
    let tcpKeepAliveInterval: Int64?
    let tcpKeepAliveIdle: Int64?
    let tcpCongestion: String?
    let tcpWindowClamp: Int64?
    let tcpMaxSeg: Int64?
    let penetrate: Bool?
    let tcpUserTimeout: Int64?
    let v6only: Bool?
    let interface: String?
    let tcpMptcp: Bool?
    let customSockopt: [CustomSockoptConfig?]?
    let addressPortStrategy: String?
    let happyEyeballs: HappyEyeballsConfig?
    let trustedXForwardedFor: [String]?
}

extension SocketConfig: Codable, Sendable {}

struct SplitHTTPConfig {
    let host: String?
    let path: String?
    let mode: String?
    let headers: [String: String]?
    let xPaddingBytes: AnyCodable?
    let xPaddingObfsMode: Bool?
    let xPaddingKey: String?
    let xPaddingHeader: String?
    let xPaddingPlacement: String?
    let xPaddingMethod: String?
    let uplinkHTTPMethod: String?
    let sessionPlacement: String?
    let sessionKey: String?
    let seqPlacement: String?
    let seqKey: String?
    let uplinkDataPlacement: String?
    let uplinkDataKey: String?
    let uplinkChunkSize: AnyCodable?
    let noGRPCHeader: Bool?
    let noSSEHeader: Bool?
    let scMaxEachPostBytes: AnyCodable?
    let scMinPostsIntervalMs: AnyCodable?
    let scMaxBufferedPosts: Int64?
    let scStreamUpServerSecs: AnyCodable?
    let serverMaxHeaderBytes: Int64?
    let xmux: XmuxConfig?
    let downloadSettings: StreamConfig?
    let extra: AnyCodable?
}

extension SplitHTTPConfig: Codable, Sendable {}

struct StrategyConfig {
    let type: String?
    let settings: AnyCodable?
}

extension StrategyConfig: Codable, Sendable {}

struct StreamConfig {
    let address: AnyCodable?
    let port: UInt64?
    let network: String?
    let security: String?
    let finalmask: FinalMask?
    let tlsSettings: TLSConfig?
    let realitySettings: REALITYConfig?
    let rawSettings: TCPConfig?
    let tcpSettings: TCPConfig?
    let xhttpSettings: SplitHTTPConfig?
    let splithttpSettings: SplitHTTPConfig?
    let kcpSettings: KCPConfig?
    let grpcSettings: GRPCConfig?
    let wsSettings: WebSocketConfig?
    let httpupgradeSettings: HttpUpgradeConfig?
    let hysteriaSettings: HysteriaConfig?
    let sockopt: SocketConfig?
}

extension StreamConfig: Codable, Sendable {}

struct SystemPolicy {
    let statsInboundUplink: Bool?
    let statsInboundDownlink: Bool?
    let statsOutboundUplink: Bool?
    let statsOutboundDownlink: Bool?
}

extension SystemPolicy: Codable, Sendable {}

struct TCPConfig {
    let header: AnyCodable?
    let acceptProxyProtocol: Bool?
}

extension TCPConfig: Codable, Sendable {}

struct TLSCertConfig {
    let certificateFile: String?
    let certificate: [String]?
    let keyFile: String?
    let key: [String]?
    let usage: String?
    let ocspStapling: UInt64?
    let oneTimeLoading: Bool?
    let buildChain: Bool?
}

extension TLSCertConfig: Codable, Sendable {}

struct TLSConfig {
    let allowInsecure: Bool?
    let certificates: [TLSCertConfig?]?
    let serverName: String?
    let alpn: [String]?
    let enableSessionResumption: Bool?
    let disableSystemRoot: Bool?
    let minVersion: String?
    let maxVersion: String?
    let cipherSuites: String?
    let fingerprint: String?
    let rejectUnknownSni: Bool?
    let curvePreferences: [String]?
    let masterKeyLog: String?
    let pinnedPeerCertSha256: String?
    let verifyPeerCertByName: String?
    let verifyPeerCertInNames: [String]?
    let echServerKeys: String?
    let echConfigList: String?
    let echForceQuery: String?
    let echSockopt: SocketConfig?
}

extension TLSConfig: Codable, Sendable {}

struct UdpHop {
    let ports: AnyCodable?
    let interval: AnyCodable?
}

extension UdpHop: Codable, Sendable {}

struct VersionConfig {
    let min: String?
    let max: String?
}

extension VersionConfig: Codable, Sendable {}

struct WebSocketConfig {
    let host: String?
    let path: String?
    let headers: [String: String]?
    let acceptProxyProtocol: Bool?
    let heartbeatPeriod: UInt64?
}

extension WebSocketConfig: Codable, Sendable {}

struct XmuxConfig {
    let maxConcurrency: AnyCodable?
    let maxConnections: AnyCodable?
    let cMaxReuseTimes: AnyCodable?
    let hMaxRequestTimes: AnyCodable?
    let hMaxReusableSecs: AnyCodable?
    let hKeepAlivePeriod: Int64?
}

extension XmuxConfig: Codable, Sendable {}

struct healthCheckSettings {
    let destination: String?
    let connectivity: String?
    let interval: Int64?
    let sampling: Int64?
    let timeout: Int64?
    let httpMethod: String?
}

extension healthCheckSettings: Codable, Sendable {}
