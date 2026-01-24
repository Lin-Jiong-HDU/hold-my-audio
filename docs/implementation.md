# Hold My Audio - 实现流程

## 概述
这是一个 Go 语言实现的互动式 AI 播客引擎，核心功能是**打断-响应-适应**循环。

---

## 实现步骤

### 第一步：定义核心接口

**文件：** `internal/ai/types.go`

```go
type LLMEngine interface {
    GenerateStream(ctx context.Context, prompt string) <-chan string
    GenerateResponse(ctx context.Context, question, context string) <-chan string
}

type TTSEngine interface {
    SynthesizeStream(ctx context.Context, text string) <-chan []byte
}

type VADMonitor interface {
    Start(ctx context.Context) <-chan struct{} // 检测到语音时发送信号
}

type AudioPlayer interface {
    PlayStream(ctx context.Context, audioStream <-chan []byte) error
    Stop()
}

type AudioRecorder interface {
    Record(ctx context.Context) (string, error) // 返回识别的文本
}
```

---

### 第二步：实现状态机（Orchestrator）

**文件：** `internal/orchestrator/state.go`

```go
type State int

const (
    IDLE State = iota
    PLAYING
    INTERRUPTED
    THINKING
    UPDATING
)

type Orchestrator struct {
    state      State
    stateCtx   context.Context
    cancelFunc context.CancelFunc
    llm        LLMEngine
    tts        TTSEngine
    vad        VADMonitor
    player     AudioPlayer
    recorder   AudioRecorder
}
```

**状态转换逻辑：**
- `IDLE -> PLAYING`: 用户输入主题，开始生成播客
- `PLAYING -> INTERRUPTED`: VAD 检测到用户语音
- `INTERRUPTED -> THINKING`: 用户提问完成，开始生成回答
- `THINKING -> PLAYING`: 回答播放完成，恢复播客
- `THINKING -> UPDATING`: 需要调整剩余脚本

---

### 第三步：实现主循环

**文件：** `internal/orchestrator/orchestrator.go`

```go
func (o *Orchestrator) Start(topic string) error {
    // 1. 生成初始播客脚本
    scriptStream := o.llm.GenerateStream(ctx, topic)

    // 2. 启动播放循环
    go o.playbackLoop(scriptStream)

    // 3. 监听打断
    go o.monitorInterruption()

    return nil
}

func (o *Orchestrator) playbackLoop(scriptStream <-chan string) {
    for {
        select {
        case text := <-scriptStream:
            audioStream := o.tts.SynthesizeStream(ctx, text)
            o.player.PlayStream(ctx, audioStream)
        case <-o.vad.Start(ctx):
            o.handleInterruption()
        }
    }
}

func (o *Orchestrator) handleInterruption() {
    o.cancelFunc() // 停止当前播放

    question, _ := o.recorder.Record(ctx)
    responseStream := o.llm.GenerateResponse(ctx, question, context)

    // 播放回答
    for text := range responseStream {
        audioStream := o.tts.SynthesizeStream(ctx, text)
        o.player.PlayStream(ctx, audioStream)
    }
}
```

---

### 第四步：实现各模块（最小化版本）

#### 4.1 LLM 模块 (`internal/llm/openai.go`)
- 使用 OpenAI API
- 实现流式输出

#### 4.2 TTS 模块 (`internal/tts/openai.go`)
- 使用 OpenAI TTS API
- 返回音频流

#### 4.3 VAD 模块 (`internal/vad/silero.go`)
- 使用 Silero VAD 模型
- 检测语音活动

#### 4.4 Audio 模块 (`internal/audio/player.go`, `recorder.go`)
- Player: 使用 `oto` 库播放音频
- Recorder: 使用 OpenAI Whisper API 进行 STT

---

### 第五步：主程序入口

**文件：** `cmd/hma/main.go`

```go
func main() {
    llm := llm.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
    tts := tts.NewOpenAI(os.Getenv("OPENAI_API_KEY"))
    vad := vad.NewSilero()
    player := audio.NewPlayer()
    recorder := audio.NewWhisperRecorder(os.Getenv("OPENAI_API_KEY"))

    orch := orchestrator.New(llm, tts, vad, player, recorder)

    topic := flag.String("topic", "", "Podcast topic")
    flag.Parse()

    orch.Start(*topic)
}
```

---

## 依赖库

```go
require (
    github.com/sashabaranov/go-openai v1.20.4  // OpenAI API
    github.com/hajimehoshi/oto v2.4.2          // 音频播放
    github.com/buger/goterm/v3                 // 终端输出（可选）
)
```

---

## 运行

```bash
go run cmd/hma/main.go --topic "The future of AI"
```

---

## 核心要点

1. **Context 取消**: 使用 `context.WithCancel` 实现立即停止播放
2. **流式处理**: 所有数据传输使用 `chan`，避免等待完整生成
3. **状态驱动**: 所有行为由状态机控制，便于扩展
4. **最小化接口**: 每个模块只暴露必要的方法
