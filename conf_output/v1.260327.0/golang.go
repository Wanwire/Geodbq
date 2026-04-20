package main

import (
  "encoding/json"
  "github.com/xtls/xray-core/infra/conf/cfgcommon/duration"
)

type APIConfig struct {
  Tag string `json:"tag"`
  Listen string `json:"listen"`
  Services []string `json:"services"`
}

type BalancingRule struct {
  Tag string `json:"tag"`
  Selectors []string `json:"selector"`
  Strategy StrategyConfig `json:"strategy"`
  FallbackTag string `json:"fallbackTag"`
}

type BridgeConfig struct {
  Tag string `json:"tag"`
  Domain string `json:"domain"`
}

type BurstObservatoryConfig struct {
  SubjectSelector []string `json:"subjectSelector"`
  HealthCheck *healthCheckSettings `json:"pingConfig,omitempty"`
}

type Config struct {
  Transport map[string]json.RawMessage `json:"transport"`
  LogConfig *LogConfig `json:"log"`
  RouterConfig *RouterConfig `json:"routing"`
  DNSConfig *DNSConfig `json:"dns"`
  InboundConfigs []InboundDetourConfig `json:"inbounds"`
  OutboundConfigs []OutboundDetourConfig `json:"outbounds"`
  Policy *PolicyConfig `json:"policy"`
  API *APIConfig `json:"api"`
  Metrics *MetricsConfig `json:"metrics"`
  Stats json.RawMessage `json:"stats"`
  Reverse *ReverseConfig `json:"reverse"`
  FakeDNS json.RawMessage `json:"fakeDns"`
  Observatory *ObservatoryConfig `json:"observatory"`
  BurstObservatory *BurstObservatoryConfig `json:"burstObservatory"`
  Version *VersionConfig `json:"version"`
}

type CustomSockoptConfig struct {
  Syetem string `json:"system"`
  Network string `json:"network"`
  Level string `json:"level"`
  Opt string `json:"opt"`
  Value string `json:"value"`
  Type string `json:"type"`
}

type DNSConfig struct {
  Servers []*NameServerConfig `json:"servers"`
  Hosts json.RawMessage `json:"hosts"`
  ClientIP json.RawMessage `json:"clientIp"`
  Tag string `json:"tag"`
  QueryStrategy string `json:"queryStrategy"`
  DisableCache bool `json:"disableCache"`
  ServeStale bool `json:"serveStale"`
  ServeExpiredTTL uint `json:"serveExpiredTTL"`
  DisableFallback bool `json:"disableFallback"`
  DisableFallbackIfMatch bool `json:"disableFallbackIfMatch"`
  EnableParallelQuery bool `json:"enableParallelQuery"`
  UseSystemHosts bool `json:"useSystemHosts"`
}

type FinalMask struct {
  Tcp []Mask `json:"tcp"`
  Udp []Mask `json:"udp"`
  QuicParams *QuicParamsConfig `json:"quicParams"`
}

type GRPCConfig struct {
  Authority string `json:"authority"`
  ServiceName string `json:"serviceName"`
  MultiMode bool `json:"multiMode"`
  IdleTimeout int `json:"idle_timeout"`
  HealthCheckTimeout int `json:"health_check_timeout"`
  PermitWithoutStream bool `json:"permit_without_stream"`
  InitialWindowsSize int `json:"initial_windows_size"`
  UserAgent string `json:"user_agent"`
}

type HappyEyeballsConfig struct {
  PrioritizeIPv6 bool `json:"prioritizeIPv6"`
  TryDelayMs uint `json:"tryDelayMs"`
  Interleave uint `json:"interleave"`
  MaxConcurrentTry uint `json:"maxConcurrentTry"`
}

type HttpUpgradeConfig struct {
  Host string `json:"host"`
  Path string `json:"path"`
  Headers map[string]string `json:"headers"`
  AcceptProxyProtocol bool `json:"acceptProxyProtocol"`
}

type HysteriaConfig struct {
  Version int `json:"version"`
  Auth string `json:"auth"`
  Congestion *string `json:"congestion"`
  Up *string `json:"up"`
  Down *string `json:"down"`
  UdpHop *UdpHop `json:"udphop"`
  UdpIdleTimeout int `json:"udpIdleTimeout"`
  Masquerade Masquerade `json:"masquerade"`
}

type InboundDetourConfig struct {
  Protocol string `json:"protocol"`
  PortList json.RawMessage `json:"port"`
  ListenOn json.RawMessage `json:"listen"`
  Settings *json.RawMessage `json:"settings"`
  Tag string `json:"tag"`
  StreamSetting *StreamConfig `json:"streamSettings"`
  SniffingConfig *SniffingConfig `json:"sniffing"`
}

type KCPConfig struct {
  Mtu *uint `json:"mtu"`
  Tti *uint `json:"tti"`
  UpCap *uint `json:"uplinkCapacity"`
  DownCap *uint `json:"downlinkCapacity"`
  Congestion *bool `json:"congestion"`
  ReadBufferSize *uint `json:"readBufferSize"`
  WriteBufferSize *uint `json:"writeBufferSize"`
  HeaderConfig json.RawMessage `json:"header"`
  Seed *string `json:"seed"`
}

type LogConfig struct {
  AccessLog string `json:"access"`
  ErrorLog string `json:"error"`
  LogLevel string `json:"loglevel"`
  DNSLog bool `json:"dnsLog"`
  MaskAddress string `json:"maskAddress"`
}

type Mask struct {
  Type string `json:"type"`
  Settings *json.RawMessage `json:"settings"`
}

type Masquerade struct {
  Type string `json:"type"`
  Dir string `json:"dir"`
  Url string `json:"url"`
  RewriteHost bool `json:"rewriteHost"`
  Insecure bool `json:"insecure"`
  Content string `json:"content"`
  Headers map[string]string `json:"headers"`
  StatusCode int `json:"statusCode"`
}

type MetricsConfig struct {
  Tag string `json:"tag"`
  Listen string `json:"listen"`
}

type MuxConfig struct {
  Enabled bool `json:"enabled"`
  Concurrency int `json:"concurrency"`
  XudpConcurrency int `json:"xudpConcurrency"`
  XudpProxyUDP443 string `json:"xudpProxyUDP443"`
}

type NameServerConfig struct {
  Address json.RawMessage `json:"address"`
  ClientIP json.RawMessage `json:"clientIp"`
  Port uint `json:"port"`
  SkipFallback bool `json:"skipFallback"`
  Domains []string `json:"domains"`
  ExpectedIPs []string `json:"expectedIPs"`
  ExpectIPs []string `json:"expectIPs"`
  QueryStrategy string `json:"queryStrategy"`
  Tag string `json:"tag"`
  TimeoutMs uint `json:"timeoutMs"`
  DisableCache *bool `json:"disableCache"`
  ServeStale *bool `json:"serveStale"`
  ServeExpiredTTL *uint `json:"serveExpiredTTL"`
  FinalQuery bool `json:"finalQuery"`
  UnexpectedIPs []string `json:"unexpectedIPs"`
}

type ObservatoryConfig struct {
  SubjectSelector []string `json:"subjectSelector"`
  ProbeURL string `json:"probeURL"`
  ProbeInterval duration.Duration `json:"probeInterval"`
  EnableConcurrency bool `json:"enableConcurrency"`
}

type OutboundDetourConfig struct {
  Protocol string `json:"protocol"`
  SendThrough *string `json:"sendThrough"`
  Tag string `json:"tag"`
  Settings *json.RawMessage `json:"settings"`
  StreamSetting *StreamConfig `json:"streamSettings"`
  ProxySettings *ProxyConfig `json:"proxySettings"`
  MuxSettings *MuxConfig `json:"mux"`
  TargetStrategy string `json:"targetStrategy"`
}

type Policy struct {
  Handshake *uint `json:"handshake"`
  ConnectionIdle *uint `json:"connIdle"`
  UplinkOnly *uint `json:"uplinkOnly"`
  DownlinkOnly *uint `json:"downlinkOnly"`
  StatsUserUplink bool `json:"statsUserUplink"`
  StatsUserDownlink bool `json:"statsUserDownlink"`
  StatsUserOnline bool `json:"statsUserOnline"`
  BufferSize *int `json:"bufferSize"`
}

type PolicyConfig struct {
  Levels map[uint]*Policy `json:"levels"`
  System *SystemPolicy `json:"system"`
}

type PortalConfig struct {
  Tag string `json:"tag"`
  Domain string `json:"domain"`
}

type ProxyConfig struct {
  Tag string `json:"tag"`
  TransportLayerProxy bool `json:"transportLayer"`
}

type QuicParamsConfig struct {
  Congestion string `json:"congestion"`
  Debug bool `json:"debug"`
  BrutalUp string `json:"brutalUp"`
  BrutalDown string `json:"brutalDown"`
  UdpHop UdpHop `json:"udpHop"`
  InitStreamReceiveWindow uint `json:"initStreamReceiveWindow"`
  MaxStreamReceiveWindow uint `json:"maxStreamReceiveWindow"`
  InitConnectionReceiveWindow uint `json:"initConnectionReceiveWindow"`
  MaxConnectionReceiveWindow uint `json:"maxConnectionReceiveWindow"`
  MaxIdleTimeout int `json:"maxIdleTimeout"`
  KeepAlivePeriod int `json:"keepAlivePeriod"`
  DisablePathMTUDiscovery bool `json:"disablePathMTUDiscovery"`
  MaxIncomingStreams int `json:"maxIncomingStreams"`
}

type REALITYConfig struct {
  MasterKeyLog string `json:"masterKeyLog"`
  Show bool `json:"show"`
  Target json.RawMessage `json:"target"`
  Dest json.RawMessage `json:"dest"`
  Type string `json:"type"`
  Xver uint `json:"xver"`
  ServerNames []string `json:"serverNames"`
  PrivateKey string `json:"privateKey"`
  MinClientVer string `json:"minClientVer"`
  MaxClientVer string `json:"maxClientVer"`
  MaxTimeDiff uint `json:"maxTimeDiff"`
  ShortIds []string `json:"shortIds"`
  Mldsa65Seed string `json:"mldsa65Seed"`
  LimitFallbackUpload json.RawMessage `json:"limitFallbackUpload"`
  LimitFallbackDownload json.RawMessage `json:"limitFallbackDownload"`
  Fingerprint string `json:"fingerprint"`
  ServerName string `json:"serverName"`
  Password string `json:"password"`
  PublicKey string `json:"publicKey"`
  ShortId string `json:"shortId"`
  Mldsa65Verify string `json:"mldsa65Verify"`
  SpiderX string `json:"spiderX"`
}

type ReverseConfig struct {
  Bridges []BridgeConfig `json:"bridges"`
  Portals []PortalConfig `json:"portals"`
}

type RouterConfig struct {
  RuleList json.RawMessage `json:"rules"`
  DomainStrategy *string `json:"domainStrategy"`
  Balancers []*BalancingRule `json:"balancers"`
}

type SniffingConfig struct {
  Enabled bool `json:"enabled"`
  DestOverride *[]string `json:"destOverride"`
  DomainsExcluded *[]string `json:"domainsExcluded"`
  MetadataOnly bool `json:"metadataOnly"`
  RouteOnly bool `json:"routeOnly"`
}

type SocketConfig struct {
  Mark int `json:"mark"`
  TFO interface{} `json:"tcpFastOpen"`
  TProxy string `json:"tproxy"`
  AcceptProxyProtocol bool `json:"acceptProxyProtocol"`
  DomainStrategy string `json:"domainStrategy"`
  DialerProxy string `json:"dialerProxy"`
  TCPKeepAliveInterval int `json:"tcpKeepAliveInterval"`
  TCPKeepAliveIdle int `json:"tcpKeepAliveIdle"`
  TCPCongestion string `json:"tcpCongestion"`
  TCPWindowClamp int `json:"tcpWindowClamp"`
  TCPMaxSeg int `json:"tcpMaxSeg"`
  Penetrate bool `json:"penetrate"`
  TCPUserTimeout int `json:"tcpUserTimeout"`
  V6only bool `json:"v6only"`
  Interface string `json:"interface"`
  TcpMptcp bool `json:"tcpMptcp"`
  CustomSockopt []*CustomSockoptConfig `json:"customSockopt"`
  AddressPortStrategy string `json:"addressPortStrategy"`
  HappyEyeballsSettings *HappyEyeballsConfig `json:"happyEyeballs"`
  TrustedXForwardedFor []string `json:"trustedXForwardedFor"`
}

type SplitHTTPConfig struct {
  Host string `json:"host"`
  Path string `json:"path"`
  Mode string `json:"mode"`
  Headers map[string]string `json:"headers"`
  XPaddingBytes json.RawMessage `json:"xPaddingBytes"`
  XPaddingObfsMode bool `json:"xPaddingObfsMode"`
  XPaddingKey string `json:"xPaddingKey"`
  XPaddingHeader string `json:"xPaddingHeader"`
  XPaddingPlacement string `json:"xPaddingPlacement"`
  XPaddingMethod string `json:"xPaddingMethod"`
  UplinkHTTPMethod string `json:"uplinkHTTPMethod"`
  SessionPlacement string `json:"sessionPlacement"`
  SessionKey string `json:"sessionKey"`
  SeqPlacement string `json:"seqPlacement"`
  SeqKey string `json:"seqKey"`
  UplinkDataPlacement string `json:"uplinkDataPlacement"`
  UplinkDataKey string `json:"uplinkDataKey"`
  UplinkChunkSize json.RawMessage `json:"uplinkChunkSize"`
  NoGRPCHeader bool `json:"noGRPCHeader"`
  NoSSEHeader bool `json:"noSSEHeader"`
  ScMaxEachPostBytes json.RawMessage `json:"scMaxEachPostBytes"`
  ScMinPostsIntervalMs json.RawMessage `json:"scMinPostsIntervalMs"`
  ScMaxBufferedPosts int `json:"scMaxBufferedPosts"`
  ScStreamUpServerSecs json.RawMessage `json:"scStreamUpServerSecs"`
  ServerMaxHeaderBytes int `json:"serverMaxHeaderBytes"`
  Xmux XmuxConfig `json:"xmux"`
  DownloadSettings *StreamConfig `json:"downloadSettings"`
  Extra json.RawMessage `json:"extra"`
}

type StrategyConfig struct {
  Type string `json:"type"`
  Settings *json.RawMessage `json:"settings"`
}

type StreamConfig struct {
  Address json.RawMessage `json:"address"`
  Port uint `json:"port"`
  Network *string `json:"network"`
  Security string `json:"security"`
  FinalMask *FinalMask `json:"finalmask"`
  TLSSettings *TLSConfig `json:"tlsSettings"`
  REALITYSettings *REALITYConfig `json:"realitySettings"`
  RAWSettings *TCPConfig `json:"rawSettings"`
  TCPSettings *TCPConfig `json:"tcpSettings"`
  XHTTPSettings *SplitHTTPConfig `json:"xhttpSettings"`
  SplitHTTPSettings *SplitHTTPConfig `json:"splithttpSettings"`
  KCPSettings *KCPConfig `json:"kcpSettings"`
  GRPCSettings *GRPCConfig `json:"grpcSettings"`
  WSSettings *WebSocketConfig `json:"wsSettings"`
  HTTPUPGRADESettings *HttpUpgradeConfig `json:"httpupgradeSettings"`
  HysteriaSettings *HysteriaConfig `json:"hysteriaSettings"`
  SocketSettings *SocketConfig `json:"sockopt"`
}

type SystemPolicy struct {
  StatsInboundUplink bool `json:"statsInboundUplink"`
  StatsInboundDownlink bool `json:"statsInboundDownlink"`
  StatsOutboundUplink bool `json:"statsOutboundUplink"`
  StatsOutboundDownlink bool `json:"statsOutboundDownlink"`
}

type TCPConfig struct {
  HeaderConfig json.RawMessage `json:"header"`
  AcceptProxyProtocol bool `json:"acceptProxyProtocol"`
}

type TLSCertConfig struct {
  CertFile string `json:"certificateFile"`
  CertStr []string `json:"certificate"`
  KeyFile string `json:"keyFile"`
  KeyStr []string `json:"key"`
  Usage string `json:"usage"`
  OcspStapling uint `json:"ocspStapling"`
  OneTimeLoading bool `json:"oneTimeLoading"`
  BuildChain bool `json:"buildChain"`
}

type TLSConfig struct {
  AllowInsecure bool `json:"allowInsecure"`
  Certs []*TLSCertConfig `json:"certificates"`
  ServerName string `json:"serverName"`
  ALPN *[]string `json:"alpn"`
  EnableSessionResumption bool `json:"enableSessionResumption"`
  DisableSystemRoot bool `json:"disableSystemRoot"`
  MinVersion string `json:"minVersion"`
  MaxVersion string `json:"maxVersion"`
  CipherSuites string `json:"cipherSuites"`
  Fingerprint string `json:"fingerprint"`
  RejectUnknownSNI bool `json:"rejectUnknownSni"`
  CurvePreferences *[]string `json:"curvePreferences"`
  MasterKeyLog string `json:"masterKeyLog"`
  PinnedPeerCertSha256 string `json:"pinnedPeerCertSha256"`
  VerifyPeerCertByName string `json:"verifyPeerCertByName"`
  VerifyPeerCertInNames []string `json:"verifyPeerCertInNames"`
  ECHServerKeys string `json:"echServerKeys"`
  ECHConfigList string `json:"echConfigList"`
  ECHForceQuery string `json:"echForceQuery"`
  ECHSocketSettings *SocketConfig `json:"echSockopt"`
}

type UdpHop struct {
  PortList json.RawMessage `json:"ports"`
  Interval json.RawMessage `json:"interval"`
}

type VersionConfig struct {
  MinVersion string `json:"min"`
  MaxVersion string `json:"max"`
}

type WebSocketConfig struct {
  Host string `json:"host"`
  Path string `json:"path"`
  Headers map[string]string `json:"headers"`
  AcceptProxyProtocol bool `json:"acceptProxyProtocol"`
  HeartbeatPeriod uint `json:"heartbeatPeriod"`
}

type XmuxConfig struct {
  MaxConcurrency json.RawMessage `json:"maxConcurrency"`
  MaxConnections json.RawMessage `json:"maxConnections"`
  CMaxReuseTimes json.RawMessage `json:"cMaxReuseTimes"`
  HMaxRequestTimes json.RawMessage `json:"hMaxRequestTimes"`
  HMaxReusableSecs json.RawMessage `json:"hMaxReusableSecs"`
  HKeepAlivePeriod int `json:"hKeepAlivePeriod"`
}

type healthCheckSettings struct {
  Destination string `json:"destination"`
  Connectivity string `json:"connectivity"`
  Interval duration.Duration `json:"interval"`
  SamplingCount int `json:"sampling"`
  Timeout duration.Duration `json:"timeout"`
  HttpMethod string `json:"httpMethod"`
}
