package ago

import (
	"path"

	"github.com/bincooo/ago/model"
)

type plugin struct {
	rec model.Record[string, any]
}

type innerAdapter struct {
	model.BasicAdapter

	rec model.Record[string, any]
}

func (receiver *plugin) model(mod ...string) *plugin {
	var models []model.Model
	for i := range mod {
		models = append(models, model.Model{
			Id:      mod[i],
			Object:  "model",
			Created: 1686935002,
			By:      "adapter",
		})
	}
	receiver.rec.Put("model", models)
	return receiver
}

// 上下文对话
func (receiver *plugin) Relay(yield func(ctx *model.Ctx) error) *plugin {
	receiver.rec.Put("relay", yield)
	return receiver
}

// 向量查询
func (receiver *plugin) Embed(yield func(ctx *model.Ctx) error) *plugin {
	receiver.rec.Put("embed", yield)
	return receiver
}

// 文生图
func (receiver *plugin) Image(yield func(ctx *model.Ctx) error) *plugin {
	receiver.rec.Put("image", yield)
	return receiver
}

func (receiver *plugin) Append() {
	ada := new(innerAdapter)
	ada.rec = receiver.rec
	Sdk().RegisterAdapter(ada)
}

func (receiver innerAdapter) Model() []model.Model {
	return model.JustValue[string, []model.Model](receiver.rec, "model")
}

func (receiver innerAdapter) Support(ctx *model.Ctx, mod string) bool {
	models, ok := model.GetValue[string, []model.Model](receiver.rec, "model")
	if !ok {
		return false
	}

	for _, mode := range models {
		if mode.Id == mod {
			return true
		}

		id := mode.Id
		ok, err := path.Match(id, mod)
		if err != nil {
			continue
		}

		if ok {
			return true
		}
	}
	return false
}

// 上下文对话
func (receiver innerAdapter) Relay(ctx *model.Ctx) (err error) {
	relay, ok := model.GetValue[string, func(*model.Ctx) error](receiver.rec, "relay")
	if !ok {
		return
	}
	return relay(ctx)
}

// 向量查询
func (receiver innerAdapter) Embed(ctx *model.Ctx) (err error) {
	embed, ok := model.GetValue[string, func(*model.Ctx) error](receiver.rec, "embed")
	if !ok {
		return
	}
	return embed(ctx)
}

// 文生图
func (receiver innerAdapter) Image(ctx *model.Ctx) (err error) {
	image, ok := model.GetValue[string, func(*model.Ctx) error](receiver.rec, "image")
	if !ok {
		return
	}
	return image(ctx)
}
