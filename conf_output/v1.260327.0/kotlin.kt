@Serializable(with = AnyCodable.Companion.AnyCodableSerializer::class)
class AnyCodable(val value: Any?) {
    companion object {
        object AnyCodableSerializer : KSerializer<AnyCodable> {

            @OptIn(InternalSerializationApi::class)
            override val descriptor: SerialDescriptor =
                buildSerialDescriptor("AnyCodable", SerialKind.CONTEXTUAL)

            override fun deserialize(decoder: Decoder): AnyCodable {
                val input = decoder as? JsonDecoder
                    ?: throw SerializationException("AnyCodable works only with JSON")

                val element = input.decodeJsonElement()
                return AnyCodable(decodeElement(element))
            }

            private fun decodeElement(element: JsonElement): Any? {
                return when (element) {

                    JsonNull -> null

                    is JsonPrimitive -> {
                        when {
                            element.isString -> element.content
                            element.booleanOrNull != null -> element.boolean
                            element.longOrNull != null -> element.long
                            element.doubleOrNull != null -> element.double
                            else -> element.content
                        }
                    }

                    is JsonArray -> element.map { decodeElement(it) }

                    is JsonObject -> element.mapValues { decodeElement(it.value) }
                }
            }

            override fun serialize(encoder: Encoder, value: AnyCodable) {
                val output = encoder as? JsonEncoder
                    ?: throw SerializationException("AnyCodable works only with JSON")

                output.encodeJsonElement(encodeElement(value.value))
            }

            private fun encodeElement(value: Any?): JsonElement {
                return when (value) {

                    null -> JsonNull

                    is Short -> JsonPrimitive(value)

                    is UShort -> JsonPrimitive(value)

                    is Long -> JsonPrimitive(value)

                    is ULong -> JsonPrimitive(value)

                    is Boolean -> JsonPrimitive(value)

                    is Int -> JsonPrimitive(value)

                    is Double -> JsonPrimitive(value)

                    is String -> JsonPrimitive(value)

                    is Map<*, *> -> JsonObject(
                        value.entries.associate {
                            val key = it.key as? String
                                ?: throw SerializationException("Map keys must be String")
                            key to encodeElement(it.value)
                        }
                    )

                    is List<*> -> JsonArray(
                        value.map { encodeElement(it) }
                    )

                    else -> throw SerializationException("Unsupported JSON value: ${value::class}")
                }
            }
        }
    }
}

@Serializable
data class APIConfig(
    val tag: String?
    val listen: String?
    val services: List<String>?
)

@Serializable
data class BalancingRule(
    val tag: String?
    val selector: List<String>?
    val strategy: StrategyConfig?
    val fallbackTag: String?
)

@Serializable
data class BridgeConfig(
    val tag: String?
    val domain: String?
)

@Serializable
data class BurstObservatoryConfig(
    val subjectSelector: List<String>?
    val pingConfig: healthCheckSettings?
)

@Serializable
data class Config(
    val transport: Map<String, AnyCodable>?
    val log: LogConfig?
    val routing: RouterConfig?
    val dns: DNSConfig?
    val inbounds: List<InboundDetourConfig>?
    val outbounds: List<OutboundDetourConfig>?
    val policy: PolicyConfig?
    val api: APIConfig?
    val metrics: MetricsConfig?
    val stats: AnyCodable?
    val reverse: ReverseConfig?
    val fakeDns: AnyCodable?
    val observatory: ObservatoryConfig?
    val burstObservatory: BurstObservatoryConfig?
    val version: VersionConfig?
)

@Serializable
data class CustomSockoptConfig(
    val system: String?
    val network: String?
    val level: String?
    val opt: String?
    val value: String?
    val type: String?
)

@Serializable
data class DNSConfig(
    val servers: List<NameServerConfig?>?
    val hosts: AnyCodable?
    val clientIp: AnyCodable?
    val tag: String?
    val queryStrategy: String?
    val disableCache: Boolean?
    val serveStale: Boolean?
    val serveExpiredTTL: Long?
    val disableFallback: Boolean?
    val disableFallbackIfMatch: Boolean?
    val enableParallelQuery: Boolean?
    val useSystemHosts: Boolean?
)

@Serializable
data class FinalMask(
    val tcp: List<Mask>?
    val udp: List<Mask>?
    val quicParams: QuicParamsConfig?
)

@Serializable
data class GRPCConfig(
    val authority: String?
    val serviceName: String?
    val multiMode: Boolean?
    val idle_timeout: Long?
    val health_check_timeout: Long?
    val permit_without_stream: Boolean?
    val initial_windows_size: Long?
    val user_agent: String?
)

@Serializable
data class HappyEyeballsConfig(
    val prioritizeIPv6: Boolean?
    val tryDelayMs: Long?
    val interleave: Long?
    val maxConcurrentTry: Long?
)

@Serializable
data class HttpUpgradeConfig(
    val host: String?
    val path: String?
    val headers: Map<String, String>?
    val acceptProxyProtocol: Boolean?
)

@Serializable
data class HysteriaConfig(
    val version: Long?
    val auth: String?
    val congestion: String?
    val up: String?
    val down: String?
    val udphop: UdpHop?
    val udpIdleTimeout: Long?
    val masquerade: Masquerade?
)

@Serializable
data class InboundDetourConfig(
    val protocol: String?
    val port: AnyCodable?
    val listen: AnyCodable?
    val settings: AnyCodable?
    val tag: String?
    val streamSettings: StreamConfig?
    val sniffing: SniffingConfig?
)

@Serializable
data class KCPConfig(
    val mtu: Long?
    val tti: Long?
    val uplinkCapacity: Long?
    val downlinkCapacity: Long?
    val congestion: Boolean?
    val readBufferSize: Long?
    val writeBufferSize: Long?
    val header: AnyCodable?
    val seed: String?
)

@Serializable
data class LogConfig(
    val access: String?
    val error: String?
    val loglevel: String?
    val dnsLog: Boolean?
    val maskAddress: String?
)

@Serializable
data class Mask(
    val type: String?
    val settings: AnyCodable?
)

@Serializable
data class Masquerade(
    val type: String?
    val dir: String?
    val url: String?
    val rewriteHost: Boolean?
    val insecure: Boolean?
    val content: String?
    val headers: Map<String, String>?
    val statusCode: Long?
)

@Serializable
data class MetricsConfig(
    val tag: String?
    val listen: String?
)

@Serializable
data class MuxConfig(
    val enabled: Boolean?
    val concurrency: Long?
    val xudpConcurrency: Long?
    val xudpProxyUDP443: String?
)

@Serializable
data class NameServerConfig(
    val address: AnyCodable?
    val clientIp: AnyCodable?
    val port: Long?
    val skipFallback: Boolean?
    val domains: List<String>?
    val expectedIPs: List<String>?
    val expectIPs: List<String>?
    val queryStrategy: String?
    val tag: String?
    val timeoutMs: Long?
    val disableCache: Boolean?
    val serveStale: Boolean?
    val serveExpiredTTL: Long?
    val finalQuery: Boolean?
    val unexpectedIPs: List<String>?
)

@Serializable
data class ObservatoryConfig(
    val subjectSelector: List<String>?
    val probeURL: String?
    val probeInterval: Long?
    val enableConcurrency: Boolean?
)

@Serializable
data class OutboundDetourConfig(
    val protocol: String?
    val sendThrough: String?
    val tag: String?
    val settings: AnyCodable?
    val streamSettings: StreamConfig?
    val proxySettings: ProxyConfig?
    val mux: MuxConfig?
    val targetStrategy: String?
)

@Serializable
data class Policy(
    val handshake: Long?
    val connIdle: Long?
    val uplinkOnly: Long?
    val downlinkOnly: Long?
    val statsUserUplink: Boolean?
    val statsUserDownlink: Boolean?
    val statsUserOnline: Boolean?
    val bufferSize: Long?
)

@Serializable
data class PolicyConfig(
    val levels: Map<Long, Policy?>?
    val system: SystemPolicy?
)

@Serializable
data class PortalConfig(
    val tag: String?
    val domain: String?
)

@Serializable
data class ProxyConfig(
    val tag: String?
    val transportLayer: Boolean?
)

@Serializable
data class QuicParamsConfig(
    val congestion: String?
    val debug: Boolean?
    val brutalUp: String?
    val brutalDown: String?
    val udpHop: UdpHop?
    val initStreamReceiveWindow: Long?
    val maxStreamReceiveWindow: Long?
    val initConnectionReceiveWindow: Long?
    val maxConnectionReceiveWindow: Long?
    val maxIdleTimeout: Long?
    val keepAlivePeriod: Long?
    val disablePathMTUDiscovery: Boolean?
    val maxIncomingStreams: Long?
)

@Serializable
data class REALITYConfig(
    val masterKeyLog: String?
    val show: Boolean?
    val target: AnyCodable?
    val dest: AnyCodable?
    val type: String?
    val xver: Long?
    val serverNames: List<String>?
    val privateKey: String?
    val minClientVer: String?
    val maxClientVer: String?
    val maxTimeDiff: Long?
    val shortIds: List<String>?
    val mldsa65Seed: String?
    val limitFallbackUpload: AnyCodable?
    val limitFallbackDownload: AnyCodable?
    val fingerprint: String?
    val serverName: String?
    val password: String?
    val publicKey: String?
    val shortId: String?
    val mldsa65Verify: String?
    val spiderX: String?
)

@Serializable
data class ReverseConfig(
    val bridges: List<BridgeConfig>?
    val portals: List<PortalConfig>?
)

@Serializable
data class RouterConfig(
    val rules: List<AnyCodable>?
    val domainStrategy: String?
    val balancers: List<BalancingRule?>?
)

@Serializable
data class SniffingConfig(
    val enabled: Boolean?
    val destOverride: List<String>?
    val domainsExcluded: List<String>?
    val metadataOnly: Boolean?
    val routeOnly: Boolean?
)

@Serializable
data class SocketConfig(
    val mark: Long?
    val tcpFastOpen: Any?
    val tproxy: String?
    val acceptProxyProtocol: Boolean?
    val domainStrategy: String?
    val dialerProxy: String?
    val tcpKeepAliveInterval: Long?
    val tcpKeepAliveIdle: Long?
    val tcpCongestion: String?
    val tcpWindowClamp: Long?
    val tcpMaxSeg: Long?
    val penetrate: Boolean?
    val tcpUserTimeout: Long?
    val v6only: Boolean?
    val interface: String?
    val tcpMptcp: Boolean?
    val customSockopt: List<CustomSockoptConfig?>?
    val addressPortStrategy: String?
    val happyEyeballs: HappyEyeballsConfig?
    val trustedXForwardedFor: List<String>?
)

@Serializable
data class SplitHTTPConfig(
    val host: String?
    val path: String?
    val mode: String?
    val headers: Map<String, String>?
    val xPaddingBytes: AnyCodable?
    val xPaddingObfsMode: Boolean?
    val xPaddingKey: String?
    val xPaddingHeader: String?
    val xPaddingPlacement: String?
    val xPaddingMethod: String?
    val uplinkHTTPMethod: String?
    val sessionPlacement: String?
    val sessionKey: String?
    val seqPlacement: String?
    val seqKey: String?
    val uplinkDataPlacement: String?
    val uplinkDataKey: String?
    val uplinkChunkSize: AnyCodable?
    val noGRPCHeader: Boolean?
    val noSSEHeader: Boolean?
    val scMaxEachPostBytes: AnyCodable?
    val scMinPostsIntervalMs: AnyCodable?
    val scMaxBufferedPosts: Long?
    val scStreamUpServerSecs: AnyCodable?
    val serverMaxHeaderBytes: Long?
    val xmux: XmuxConfig?
    val downloadSettings: StreamConfig?
    val extra: AnyCodable?
)

@Serializable
data class StrategyConfig(
    val type: String?
    val settings: AnyCodable?
)

@Serializable
data class StreamConfig(
    val address: AnyCodable?
    val port: Long?
    val network: String?
    val security: String?
    val finalmask: FinalMask?
    val tlsSettings: TLSConfig?
    val realitySettings: REALITYConfig?
    val rawSettings: TCPConfig?
    val tcpSettings: TCPConfig?
    val xhttpSettings: SplitHTTPConfig?
    val splithttpSettings: SplitHTTPConfig?
    val kcpSettings: KCPConfig?
    val grpcSettings: GRPCConfig?
    val wsSettings: WebSocketConfig?
    val httpupgradeSettings: HttpUpgradeConfig?
    val hysteriaSettings: HysteriaConfig?
    val sockopt: SocketConfig?
)

@Serializable
data class SystemPolicy(
    val statsInboundUplink: Boolean?
    val statsInboundDownlink: Boolean?
    val statsOutboundUplink: Boolean?
    val statsOutboundDownlink: Boolean?
)

@Serializable
data class TCPConfig(
    val header: AnyCodable?
    val acceptProxyProtocol: Boolean?
)

@Serializable
data class TLSCertConfig(
    val certificateFile: String?
    val certificate: List<String>?
    val keyFile: String?
    val key: List<String>?
    val usage: String?
    val ocspStapling: Long?
    val oneTimeLoading: Boolean?
    val buildChain: Boolean?
)

@Serializable
data class TLSConfig(
    val allowInsecure: Boolean?
    val certificates: List<TLSCertConfig?>?
    val serverName: String?
    val alpn: List<String>?
    val enableSessionResumption: Boolean?
    val disableSystemRoot: Boolean?
    val minVersion: String?
    val maxVersion: String?
    val cipherSuites: String?
    val fingerprint: String?
    val rejectUnknownSni: Boolean?
    val curvePreferences: List<String>?
    val masterKeyLog: String?
    val pinnedPeerCertSha256: String?
    val verifyPeerCertByName: String?
    val verifyPeerCertInNames: List<String>?
    val echServerKeys: String?
    val echConfigList: String?
    val echForceQuery: String?
    val echSockopt: SocketConfig?
)

@Serializable
data class UdpHop(
    val ports: AnyCodable?
    val interval: AnyCodable?
)

@Serializable
data class VersionConfig(
    val min: String?
    val max: String?
)

@Serializable
data class WebSocketConfig(
    val host: String?
    val path: String?
    val headers: Map<String, String>?
    val acceptProxyProtocol: Boolean?
    val heartbeatPeriod: Long?
)

@Serializable
data class XmuxConfig(
    val maxConcurrency: AnyCodable?
    val maxConnections: AnyCodable?
    val cMaxReuseTimes: AnyCodable?
    val hMaxRequestTimes: AnyCodable?
    val hMaxReusableSecs: AnyCodable?
    val hKeepAlivePeriod: Long?
)

@Serializable
data class healthCheckSettings(
    val destination: String?
    val connectivity: String?
    val interval: Long?
    val sampling: Long?
    val timeout: Long?
    val httpMethod: String?
)
