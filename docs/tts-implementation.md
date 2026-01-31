# TTS 模块实现方案

## 1. 现有接口

项目已定义 `TTSEngine` 接口：

```go
type TTSEngine interface {
    SynthesizeStream(ctx context.Context, text string) <-chan []byte
}
```

## 2. 设计思路

### 2.1 是否需要兼容层？

**不需要复杂的转译层**，原因：
- 已有的 `TTSEngine` 接口足够简单，只有一个方法
- 不同 TTS 服务内部实现差异很大，强行统一参数只会增加复杂度
- 每个服务的配置参数直接通过构造函数传入即可

### 2.2 支持多种 TTS 的方式

```
internal/tts/
├── types.go       # 接口定义（已存在）
├── openai.go      # OpenAI TTS 实现（已有框架）
├── glm.go         # GLM-TTS 实现（新增）
├── edge.go        # Edge-TTS 实现（可选）
└── local.go       # 本地模型如 Piper（可选）
```

每个实现只需要：
1. 实现自己的 `NewXXX(config)` 构造函数
2. 实现 `SynthesizeStream()` 方法
3. 内部处理各自 API 的差异

## 3. GLM-TTS 实现要点

### 3.1 API 规格总结

- **端点**: `POST https://open.bigmodel.cn/api/paas/v4/audio/speech`
- **认证**: `Authorization: Bearer <api_key>`
- **请求体**:
  ```json
  {
    "model": "glm-tts",
    "input": "要转换的文本",
    "voice": "tongtong",
    "response_format": "pcm",
    "stream": true,
    "speed": 1.0,
    "volume": 1.0,
    "encode_format": "base64"
  }
  ```
- **流式响应**: Event Stream 格式，每块是 base64 编码的 PCM 数据

### 3.2 实现步骤

1. 发送 HTTP 请求，`stream: true`
2. 读取 SSE (Server-Sent Events) 流
3. 解析每行的 base64 数据
4. 解码后写入 channel
5. 处理 context 取消

### 3.3 简单代码结构

```go
type GLM struct {
    apiKey  string
    voice   string
    speed   float64
    volume  float64
}

func NewGLM(apiKey string) *GLM {
    return &GLM{
        apiKey:  apiKey,
        voice:   "tongtong",
        speed:   1.0,
        volume:  1.0,
    }
}

func (g *GLM) SynthesizeStream(ctx context.Context, text string) <-chan []byte {
    ch := make(chan []byte)
    go func() {
        defer close(ch)
        // HTTP POST 请求
        // 读取流式响应
        // 解析 base64 数据
        // 发送到 channel
    }()
    return ch
}
```

## 4. 使用示例

```go
// 创建 GLM TTS 实例
tts := tts.NewGLM("your-api-key")

// 合成语音流
audioStream := tts.SynthesizeStream(ctx, "你好，欢迎收听播客")

// 播放
player.PlayStream(ctx, audioStream)
```

## 5. 扩展其他 TTS

同样的模式可以用于：
- **OpenAI TTS**: `internal/tts/openai.go`
- **Edge-TTS**: `internal/tts/edge.go`（调用本地 edge-tts 命令）
- **Piper**: `internal/tts/piper.go`（本地 TTS 引擎）

每个实现只需遵循 `TTSEngine` 接口，内部可以完全不同。
