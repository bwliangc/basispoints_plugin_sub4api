# BasisPoints Transport plugin for Sub4API 1.1.4 — v0.2.2

本版是在 v0.2.1 基础上，结合 `JaxsonWang/cpa-plugin-oai-basispoints` v0.1.9、当前 OpenAI Codex 的 Responses Lite / multi-agent 请求形状，以及现网报错继续修正。

## v0.2.2 修复

- 修复多代理历史：当前 Codex 会把代理间消息放成私有 `agent_message` item，并携带顶层 `author` / `recipient`。BasisPoints 不接受这些字段，会报 `Unknown parameter: input[N].author`。本版把可见文本安全降级成普通 `assistant` message，并去掉私有路由字段。
- 修复“思考过程显示两遍”：旧版只要发生工具桥接，就从 `response.completed` 重建整条 SSE，导致 reasoning item 以完整内容同时出现在 `output_item.added` 和 `output_item.done`。本版不再重建整条流，而是保留 BasisPoints 原始 reasoning / message SSE，只过滤原生 `run_officejs` 工具事件并在结束前替换为客户端真实工具事件。
- 保留 v0.2.1 的 Responses Lite 工具识别：同时扫描顶层 `tools` 与 `input[].additional_tools`。
- `additional_tools` 只用于建立客户端工具目录，不发送给 BasisPoints。
- function/custom 工具桥接继续通过 `run_officejs`；function 参数同时兼容对象和 JSON 字符串，custom 工具兼容 `input` / `args`。
- 路由日志新增 `agent_messages` 计数，便于判断多代理历史是否被兼容层处理。

## 与 cpa-plugin-oai-basispoints 的关系

参考项目确认了 BasisPoints 的核心协议边界：客户端工具不能原样发给 BPS，需要改成 developer 工具目录，再通过服务端原生 `run_officejs` 中转；请求还需要 `model_selection=explicit`、`store=false`、`reasoning_effort`、稳定的 `task_id` / `turn_id` / `agent_iteration` 以及 Excel/BasisPoints profile headers。

本插件没有直接照搬参考项目的两处行为：

1. 参考项目 v0.1.9 的 `translateInputItems` 目前没有单独处理最新 Codex 的 `agent_message`，未知 item 会原样放行，因此对当前多代理历史同样存在 `author` 字段被 BPS 拒绝的风险；本版补了兼容转换。
2. 参考项目的 `syntheticStream` 也是把最终 response 重新合成为 SSE。针对你实际出现的 reasoning 重复问题，本版改为“保留原流，只替换工具事件”，避免重复展示。

## 默认设置

- endpoint: `https://bps.openai.com/basispoints/api/responses`
- models: `gpt-6-astra`, `gpt-5.6-sol`
- auth mode: `chatgpt`
- 工具兼容: `auto`（推荐）

> BasisPoints 是未公开文档化的内部 endpoint，行为可能变化。插件不会记录 access token、账号 ID、prompt 或工具输出正文。

## 升级时保留原签名密钥

最推荐：把旧源码目录中的 `.publisher-key.pem` 复制到本目录：

```bash
cp /旧版目录/.publisher-key.pem ./.publisher-key.pem
chmod 600 .publisher-key.pem
```

或者构建时指定：

```bash
PUBLISHER_KEY_FILE=/旧版目录/.publisher-key.pem ./build.sh
```

这样 v0.2.2 会继续使用你已经写入 Sub4API `config.yaml` 的 `basispoints-local-v1` 公钥，通常不需要再次改 `trusted_publishers`。

如果不保留旧私钥，`build.sh` 会生成新的 Ed25519 密钥，这时必须把新的 `dist/trusted-publisher.yaml` 公钥更新进 Sub4API 配置并重启宿主。

## 构建

```bash
chmod +x build.sh
./build.sh
```

飞牛 NAS 的 `/home/fnroot` 不存在也没关系，构建缓存会自动放在源码目录的 `.build-state/`。

生成：

```text
dist/basispoints-transport-0.2.2.s2plugin
dist/trusted-publisher.yaml
.publisher-key.pem
```

## 升级安装

Sub4API 不允许在旧插件仍启用时上传同 ID 的新版本：

1. 后台 → 插件管理 → `BasisPoints Transport` → **停用**。
2. 直接点“安装插件”，上传 `basispoints-transport-0.2.2.s2plugin`。
3. 打开配置：BasisPoints 路由开；Models 保持 `gpt-6-astra` / `gpt-5.6-sol`；工具兼容选 **自动桥接**。
4. 保存，OAuth 流量比例设为 100%，再启用。

## 日志

普通 Astra 请求：

```text
[basispoints] route model=gpt-6-astra tools=8 top_level=0 additional=8 agent_messages=0 callable=8 bridge=auto endpoint=https://bps.openai.com/basispoints/api/responses
```

发生多代理历史后，可能看到：

```text
[basispoints] route model=gpt-6-astra tools=8 top_level=0 additional=8 agent_messages=2 callable=8 bridge=auto endpoint=https://bps.openai.com/basispoints/api/responses
```

模型要求调用客户端工具时：

```text
[basispoints] bridged tool call call_id=call_xxxxxx… tool=collaboration.spawn_agent
```

## 回退

后台插件管理直接“停用”即可恢复 Sub4API 原生 OpenAI OAuth 路径。配置里关闭“BasisPoints 路由”也会让请求纯透传原上游。
