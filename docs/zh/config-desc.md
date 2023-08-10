# 配置文件解析

默认在～/.sophon-gateway/config.toml。

```toml
# 被代理线上组件的服务地址
# 可选
Node = "/dns/node/tcp/3453"
Messager = "/dns/messager/tcp/39812"
Droplet = "/dns/market/tcp/41235"
Miner = "/dns/miner/tcp/12308"


[API]
  ListenAddress = "/ip4/127.0.0.1/tcp/45132" # 本地组件wallet和damocles-manager通过长连接和gateway保持通信

[Auth]
  Token = ""
  URL = "http://127.0.0.1:8989"

[Metrics]
  Enabled = false

  [Metrics.Exporter]
    Type = "prometheus" # 两种类型，Graphite或者Prometheus

    [Metrics.Exporter.Graphite]
      Host = "127.0.0.1"
      Namespace = "gateway"
      Port = 4569
      ReportingPeriod = "10s"

    [Metrics.Exporter.Prometheus]
      EndPoint = "/ip4/0.0.0.0/tcp/4569"
      Namespace = "gateway"
      Path = "/debug/metrics"
      RegistryType = "define"
      ReportingPeriod = "10s"

[RateLimit]
  #redis地址，用于记录用户访问的次数。如果要开启对某个user的访问限速，还需要`auth` 服务同时设置`sophon-auth user rate-limit`命令。
  Redis = "27.0.0.1:6379" 

[Trace]
  JaegerEndpoint = "localhost:6831"
  JaegerTracingEnabled = false
  ProbabilitySampler = 1.0
  ServerName = "sophon-gateway"

```
