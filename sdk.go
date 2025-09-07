package ago

import (
	"context"
	"net/http"
	"time"

	"github.com/bincooo/ago/internal"
	"github.com/bincooo/ago/internal/chromium"
	"github.com/bincooo/ago/internal/v1"
	"github.com/bincooo/ago/model"
	"github.com/bincooo/ja3"
	xtls "github.com/refraction-networking/utls"
)

type interfaces struct {
	//
}

func Sdk() interface {
	Plugin(...string) *plugin
	RegisterAdapter(adapter model.Adapter)

	Transport(proxies string) http.RoundTripper
	Env() *v1.Environ

	Chrome(ctx context.Context, proxies, userAgent, userDir string, plugins ...string) (context.Context, context.CancelFunc)

	OnInitialized(func())
	OnExited(func())
} {
	return &interfaces{}
}

func (interfaces) Plugin(mod ...string) *plugin {
	return (&plugin{rec: model.Record[string, any]{}}).model(mod...)
}

func (interfaces) RegisterAdapter(ada model.Adapter) {
	v1.AddAdapter(ada)
}

func (interfaces) Transport(proxies string) http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.IdleConnTimeout = 120 * time.Second
	return ja3.NewTransport(
		ja3.WithProxy(proxies),
		ja3.WithClientHelloID(xtls.HelloChrome_133),
		ja3.WithOriginalTransport(transport),
	)
}

func (interfaces) Chrome(ctx context.Context, proxies, userAgent, userDir string, plugins ...string) (context.Context, context.CancelFunc) {
	return chromium.InitChromium(ctx, proxies, userAgent, userDir, plugins...)
}

func (interfaces) Env() *v1.Environ {
	return v1.Env
}

func (interfaces) OnInitialized(f func()) {
	internal.AddInitialized(f)
}

func (interfaces) OnExited(f func()) {
	internal.AddExited(f)
}
