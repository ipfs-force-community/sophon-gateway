# venus-gateway metrics 使用说明

## 配置

`Metrics` 基本的配置样例如下：
```toml
[Metrics]
  # 是否开启metrics指标统计，默认为false
  Enabled = false
  
  [Metrics.Exporter]
    # 指标导出器类型，目前可选：prometheus或graphite，默认为prometheus
    Type = "prometheus"
    
    [Metrics.Exporter.Prometheus]
      # multiaddr
      EndPoint = "/ip4/0.0.0.0/tcp/4569"
      # 命名规范: "a_b_c", 不能带"-"
      Namespace = "gateway01" 
      # 指标注册表类型，可选：default（默认，会附带程序运行的环境指标）或 define（自定义）
      RegistryType = "define"
      # prometheus 服务路径
      Path = "/debug/metrics"
      # 上报周期
      ReportingPeriod = "10s"
      
    [Metrics.Exporter.Graphite]
      # 命名规范: "a_b_c", 不能带"-"
      Namespace = "gateway01" 
      # graphite exporter 收集器服务地址
      Host = "127.0.0.1"
      # graphite exporter 收集器服务监听端口
      Port = 4569
      # 上报周期
      ReportingPeriod = "10s"
```


## 导出器

目前可以选择两类导出器（`exporter`）：`Prometheus exporter` 或 `Graphite exporter`，默认是前者。

exporter端口为 `4569`，url为 `debug/metrics`, 因此对于默认的部署方式，exporter的url为 `host:4569/debug/metrics`

如果配置 `Prometheus exporter`，则在 `venus-gateway` 服务启动时会附带启动 `Prometheus exporter` 的监听服务，可以通过以下方式快速查看指标：

```bash
$ curl http://localhost:4569/debug/metrics
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="0.01"} 0
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="0.05"} 0
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="0.1"} 0
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="0.3"} 0
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="0.6"} 0
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="0.8"} 0
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="1"} 0
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="2"} 322
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="3"} 544
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="4"} 787
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="5"} 790
  gateway_wallet_sign_bucket{wallet_account="forcenet-nv16",le="6"} 791
  ... ...
```
> 如果遇到错误 `curl: (56) Recv failure: Connection reset by peer`, 请使用本机 `ip` 地址, 如下所示:
```bash
$  curl http://<ip>:4569/debug/metrics
```

如果配置 `Graphite exporter`，需要先启动 `Graphite exporter` 的收集器服务， `venus-gateway` 服务启动时将指标上报给收集器。服务启动参考 [Graphite exporter](https://github.com/prometheus/graphite_exporter) 中的说明。

`Graphite exporter` 和 `Prometheus exporter` 自身都不带图形界面的，如果需要可视化监控及更高阶的图表分析，请到 `venus-docs` 项目中查找关于 `Prometheus+Grafana` 的说明文档。


## 指标

### 钱包

```
# 钱包注册
WalletRegister   = stats.Int64("wallet_register", "Wallet register", stats.UnitDimensionless)
# 钱包注销
WalletUnregister = stats.Int64("wallet_unregister", "Wallet unregister", stats.UnitDimensionless)
# 钱包数量
WalletNum        = stats.Int64("wallet_num", "Wallet count", stats.UnitDimensionless)
# 钱包包含的地址数量
WalletAddressNum = stats.Int64("wallet_address_num", "Address owned by wallet", stats.UnitDimensionless)
# 钱包来源
WalletSource     = stats.Int64("wallet_source", "Wallet IP", stats.UnitDimensionless)
# 钱包新增地址
WalletAddAddr    = stats.Int64("wallet_add_addr", "Wallet add a new address", stats.UnitDimensionless)
# 钱包移除地址
WalletRemoveAddr = stats.Int64("wallet_remove_addr", "Wallet remove a new address", stats.UnitDimensionless)
# 钱包的连接数量
WalletConnNum    = stats.Int64("wallet_conn_num", "Wallet connection count", stats.UnitDimensionless)
```

### 矿工

```
# 矿工注册
MinerRegister   = stats.Int64("miner_register", "Miner register", stats.UnitDimensionless)
# 矿工注销
MinerUnregister = stats.Int64("miner_unregister", "Miner unregister", stats.UnitDimensionless)
# 矿工数量
MinerNum        = stats.Int64("miner_num", "Wallet count", stats.UnitDimensionless)
# 矿工来源
MinerSource     = stats.Int64("wallet_source", "Miner IP", stats.UnitDimensionless)
# 矿工的连接数量
MinerConnNum    = stats.Int64("miner_conn_num", "Miner connection count", stats.UnitDimensionless)
```

### 接口调用

```
# 签名耗时（毫秒）
WalletSign         = stats.Float64("wallet_sign", "Call WalletSign spent time", stats.UnitMilliseconds)
# 列出钱包地址耗时（毫秒）
WalletList         = stats.Float64("wallet_list", "Call WalletList spent time", stats.UnitMilliseconds)
# 计算 winnerpost 耗时（毫秒）
ComputeProof       = stats.Float64("compute_proof", "Call ComputeProof spent time", stats.UnitMilliseconds)
# 调用 IsUnsealed 耗时（毫秒）
IsUnsealed         = stats.Float64("is_unsealed", "Call IsUnsealed spent time", stats.UnitMilliseconds)
# 调用 SectorsUnsealPiece（毫秒）
SectorsUnsealPiece = stats.Float64("sectors_unseal_piece", "Call SectorsUnsealPiece spent time", stats.UnitMilliseconds)
```
