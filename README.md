# BasisPoints Transport plugin for Sub4API 1.1.4 — v0.2.2

用法，修改Sub4API 的config.ymal,最后添加
plugins:
data_dir:/app/data/plugins
allow_unsigned: false
trusted_publishers:
basispoints-local-v1: "构建出来的公钥"

如果用我编译好的，值是这个：basispoints-local-v1: "TBbYTyomVQzNEPzmTqZM/Ui24fc96EKH9hcOUl0Evk4="

自己编译的话就自己找，在生成的dist目录下有trusted-publisher.yaml。

然后在插件中心安装basispoints-transport-0.2.1.s2plugin 这个插件即可。



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
