package v1

import (
	"fmt"
	"iter"

	"github.com/bincooo/ago/logger"
	"github.com/bincooo/ago/model"
	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var (
	adapters = make([]model.Adapter, 0)
)

func AddAdapter(adapter model.Adapter) {
	adapters = append(adapters, adapter)
}

// 模型迭代器
func Models() iter.Seq[model.Model] {
	return func(yield func(model.Model) bool) {
		for _, adapter := range adapters {
			for _, mod := range adapter.Model() {
				yield(mod)
			}
		}
	}
}

// 初始化fiber api
func Initialized(addr string) {
	app := fiber.New()

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(ctx *fiber.Ctx, err interface{}) {
			logger.Sugar().Errorf("panic: %v", err)
		},
	}))

	app.Use(fiberzap.New(fiberzap.Config{
		Logger: logger.Logger(),
	}))

	app.Get("/", index)

	app.Post("v1/chat/completions", completions)
	app.Post("v1/object/completions", completions)
	app.Post("proxies/v1/chat/completions", completions)

	app.Post("/v1/embeddings", embeddings)
	app.Post("proxies/v1/embeddings", embeddings)

	app.Post("v1/images/generations", generations)
	app.Post("v1/object/generations", generations)
	app.Post("proxies/v1/images/generations", generations)

	err := app.Listen(addr)
	if err != nil {
		panic(err)
	}
}

func index(ctx *fiber.Ctx) error {
	ctx.Set("content-type", "text/html")
	return JustError(
		ctx.WriteString("<div style='color:green'>success ~</div>"),
	)
}

func completions(ctx *fiber.Ctx) (err error) {
	completion := new(model.Completion)
	if err = ctx.BodyParser(completion); err != nil {
		return
	}

	c := model.New(ctx)
	c.Type = "relay"
	c.Put("completion", completion)
	for _, adapter := range adapters {
		if !adapter.Support(c, completion.Model) {
			continue
		}
		return adapter.Relay(c)
	}

	err = writeError(ctx, fmt.Sprintf("model [%s] is not found", completion.Model))
	return
}

func embeddings(ctx *fiber.Ctx) (err error) {
	embedding := new(model.Embedding)
	if err = ctx.BodyParser(embedding); err != nil {
		return
	}

	c := model.New(ctx)
	c.Type = "embed"
	c.Put("embedding", embedding)
	for _, adapter := range adapters {
		if adapter.Support(c, embedding.Model) {
			err = adapter.Embed(c)
			break
		}
	}

	err = writeError(ctx, fmt.Sprintf("model [%s] is not found", embedding.Model))
	return
}

func generations(ctx *fiber.Ctx) (err error) {
	generation := new(model.Generation)
	if err = ctx.BodyParser(generation); err != nil {
		return
	}

	c := model.New(ctx)
	c.Type = "image"
	c.Put("generation", generation)
	for _, adapter := range adapters {
		if adapter.Support(c, generation.Model) {
			return adapter.Image(c)
		}
	}

	err = writeError(ctx, fmt.Sprintf("model [%s] is not found", generation.Model))
	return
}

func writeError(ctx *fiber.Ctx, msg string) (err error) {
	return ctx.Status(fiber.StatusInternalServerError).
		JSON(model.Record[string, any]{
			"error": msg,
		})
}
