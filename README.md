# BasisPoints Transport plugin for Sub4API 1.1.4 — v0.2.2
用法：将本次构建生成的 `dist/trusted-publisher.yaml` 中的公钥合并到 Sub4API 的 `config.yaml`：

```yaml
plugins:
  data_dir: /app/data/plugins
  allow_unsigned: false
  trusted_publishers:
    basispoints-local-v1: "本次构建出来的公钥"
```

自己编译的话就自己找，在生成的dist目录下有trusted-publisher.yaml。

然后在插件中心安装basispoints-transport-0.2.2.s2plugin 这个插件即可。

## v0.2.2 指定账号路由

在插件配置页的「指定账号 ID」中填写需要走 BasisPoints 的 **Sub4API 数字账号 ID**，每行一个，也支持逗号或空格分隔。这里不是 ChatGPT 账号 ID、邮箱、用户 ID 或 API Key。

只有「启用路由 + 账号 ID 在列表中 + 模型匹配」同时满足时，插件才转换请求并转发到 BasisPoints。其他请求保留原 URL、请求头、请求体并通过原上游转发。

- 账号筛选使用宿主协议的 `ForwardRequestStart.account_id`，不从请求头推断。
- 列表为空、旧配置缺少 `account_ids` 或请求没有有效账号 ID 时，全部走原上游。
- **从 v0.2.1 升级后需要先填写账号 ID 并保存，才会恢复指定账号的 BasisPoints 路由。**
- JSON 配置字段示例：`"account_ids": [12, 34]`。ID 必须是 1 到 9007199254740991 的整数，重复项会自动合并。
- 这只改变插件收到请求后的路由，不改变 Sub4API 的账号调度或插件挂载范围。

## v0.2.1 修复

- 修复 GPT-6 Astra / Responses Lite 工具识别：同时扫描顶层 `tools` 与 `input[].additional_tools`。
- `additional_tools` 只用于建立客户端工具目录，不再原样发送给 BasisPoints。
- 兼容 `function: {name, parameters...}` 的嵌套函数声明。
- 路由日志新增 `top_level`、`additional`、`callable` 计数，方便确认桥接是否真正看到 Codex 工具。


这个版本用于修复 v0.1.0 在 BasisPoints 上出现的 `422: Invalid request body`。

## v0.2.1 关键变化

1. 不再把客户端 `tools` 原样发送给 BasisPoints。
2. 自动把客户端工具目录写入 developer 消息。
3. 让模型通过 BasisPoints 原生注入的 `run_officejs` 作为传输工具。
4. 插件拦截 `run_officejs`，把其中 `code` 的嵌套 JSON 转回标准 `function_call` / `custom_tool_call` 给客户端执行。
5. 下一轮会把同一 `call_id` 对应的客户端工具结果重新挂回原来的 `run_officejs` 身份。
6. 为同一个用户 turn 生成稳定 `turn_id`，工具迭代只增加 `agent_iteration`。
7. 把 Codex/Responses 请求收敛到 BasisPoints 已知可接受的字段集合，并把 `max` / `ultra` reasoning effort 映射到 `xhigh`。
8. 增加 Excel/BasisPoints client profile 请求头。
9. Docker 日志中会出现不含 token/提示词的路由与工具桥接日志。

> BasisPoints 是未公开文档化的内部 endpoint，行为可能变化。插件不会记录 access token、账号 ID 或提示词。

## 默认设置

- endpoint: `https://bps.openai.com/basispoints/api/responses`
- models: `gpt-6-astra`, `gpt-5.6-sol`
- auth mode: `chatgpt`
- 指定账号: `[]`（默认不转发任何账号到 BasisPoints）
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

需要 Go 1.24+（当前依赖要求）、Python 3、curl 和支持 Ed25519 的 OpenSSL。

```bash
chmod +x build.sh
./build.sh
```

## 回退

后台插件管理直接“停用”即可恢复 Sub4API 原生 OpenAI OAuth 路径。配置里关闭“BasisPoints 路由”也会让请求纯透传原上游。
