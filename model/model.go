package model

import (
	"encoding/json"
	"iter"
	"maps"
	"reflect"

	"github.com/bincooo/ago/kit"
	"github.com/bincooo/ago/stream"
)

type Adapter interface {
	// 判定函数
	Support(ctx *Ctx, model string) bool
	// 上下文对话
	Relay(ctx *Ctx) error
	// 向量查询
	Embed(ctx *Ctx) error
	// 文生图
	Image(ctx *Ctx) error
	// 模型列表
	Model() []Model
}

type BasicAdapter struct {
}

func (BasicAdapter) Relay(*Ctx) error {
	return nil
}

func (BasicAdapter) Embed(*Ctx) error {
	return nil
}

func (BasicAdapter) Image(*Ctx) error {
	return nil
}

type Model struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	By      string `json:"owned_by"`
}

type Completion struct {
	System        string              `json:"system,omitempty"`
	Messages      []CompletionMessage `json:"messages"`
	Tools         []CompletionTool    `json:"tools,omitempty"`
	Model         string              `json:"model,omitempty"`
	MaxTokens     int                 `json:"max_tokens"`
	StopSequences []string            `json:"stop,omitempty"`
	Temperature   float32             `json:"temperature"`
	TopK          int                 `json:"top_k,omitempty"`
	TopP          float32             `json:"top_p,omitempty"`
	Stream        bool                `json:"stream,omitempty"`
	ToolChoice    interface{}         `json:"tool_choice,omitempty"`
}

type CompletionMessage = Record[string, any]
type CompletionTool = Record[string, any]

type Generation struct {
	Model   string `json:"model"`
	Message string `json:"prompt"`
	N       int    `json:"n"`
	Size    string `json:"size"`
	Style   string `json:"style"`
	Quality string `json:"quality"`
}

type Embedding struct {
	Input          interface{} `json:"input"`
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
	Dimensions     int         `json:"dimensions,omitempty"`
	User           string      `json:"user,omitempty"`
}

type Response struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
	Usage ResponseUsage `json:"usage,omitempty"`
}

type ResponseUsage Record[string, any]

type Choice struct {
	Index   int `json:"index"`
	Message *struct {
		Role             string `json:"role,omitempty"`
		Content          string `json:"content,omitempty"`
		ReasoningContent string `json:"reasoning_content,omitempty"`

		ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
	} `json:"message,omitempty"`
	Delta *struct {
		Type             string `json:"type,omitempty"`
		Role             string `json:"role,omitempty"`
		Content          string `json:"content,omitempty"`
		ReasoningContent string `json:"reasoning_content,omitempty"`

		ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
	} `json:"delta,omitempty"`
	FinishReason *string `json:"finish_reason"`
}

type ChoiceToolCall Record[string, any]

type Record[Key comparable, Value any] map[Key]Value

func (rec Record[Key, Value]) Put(k Key, v Value) {
	rec[k] = v
}

// 获取值
func (rec Record[Key, Value]) Get(k Key) Value {
	return rec[k]
}

// 删除值
func (rec Record[Key, Value]) Del(k Key) {
	delete(rec, k)
}

// 元素数量
func (rec Record[Key, Value]) Len() int {
	return len(rec)
}

// keys 迭代器
func (rec Record[Key, Value]) Keys() iter.Seq[Key] {
	return maps.Keys(rec)
}

// values 迭代器
func (rec Record[Key, Value]) Values() iter.Seq[Value] {
	return maps.Values(rec)
}

// 是否包含 key
func (rec Record[Key, Value]) Contains(k Key) bool {
	value := rec[k]
	return stream.NotNil[Value]()(value)
}

// 深克隆
func (rec Record[Key, Value]) Clone() Record[Key, Value] {
	return kit.Copy(rec)
}

// 字符串序列化
func (rec Record[Key, Value]) String() string {
	chunk, err := json.Marshal(rec)
	if err != nil {
		panic(err)
	}
	return string(chunk)
}

// 值比较
func (rec Record[Key, Value]) ValueEqual(k Key, v Value) (ok bool) {
	if !rec.Contains(k) {
		return
	}

	return reflect.DeepEqual(v, rec.Get(k))
}

// 值包含
func (rec Record[Key, Value]) ValueEquals(k Key, values ...Value) (ok bool) {
	if !rec.Contains(k) {
		return
	}

	for _, value := range values {
		if rec.ValueEqual(k, value) {
			return true
		}
	}
	return
}

// 获取值
//
//	@param rec Record实例
//	@param k 实例的key值
func JustValue[Key comparable, Value any](rec Record[Key, any], k Key) (value Value) {
	value, _ = GetValue[Key, Value](rec, k)
	return
}

// 获取值
//
//	@param rec Record实例
//	@param k 实例的key值
func GetValue[Key comparable, Value any](rec Record[Key, any], k Key) (value Value, ok bool) {
	return get[Key, Value](rec, k)
}

func get[Key comparable, Value any](rec Record[Key, any], k Key) (value Value, ok bool) {
	if rec == nil || !rec.Contains(k) {
		return
	}

	value, ok = rec.Get(k).(Value)
	if !ok {
		return
	}

	return
}
