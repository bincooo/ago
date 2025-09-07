package chromium

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/bincooo/ago/internal/chromium/plugins"
	"github.com/bincooo/ago/logger"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

func Run(ctx context.Context, actions ...chromedp.Action) error {
	return chromedp.Run(ctx, actions...)
}

func TaskLogger(message string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) (_ error) {
			logger.Sugar().Info(message)
			return
		}),
	}
}

func Goto(urlstr string) chromedp.NavigateAction {
	return chromedp.Navigate(urlstr)
}

func Input(selector, value string) chromedp.QueryAction {
	return chromedp.SendKeys(selector, value, chromedp.ByQuery)
}

func Click(selector string) chromedp.QueryAction {
	return chromedp.Click(selector)
}

func Sleep(d time.Duration) chromedp.Action {
	return chromedp.Sleep(d)
}

func WaitVisible(selector string) chromedp.QueryAction {
	return chromedp.WaitVisible(selector)
}

func MouseDragNode(n *cdp.Node, cxt context.Context) error {
	boxes, err := dom.GetContentQuads().WithNodeID(n.NodeID).Do(cxt)
	if err != nil {
		return err
	}
	if len(boxes) == 0 {
		return chromedp.ErrInvalidDimensions
	}
	content := boxes[0]
	c := len(content)
	if c%2 != 0 || c < 1 {
		return chromedp.ErrInvalidDimensions
	}
	var x, y float64
	for i := 0; i < c; i += 2 {
		x += content[i]
		y += content[i+1]
	}
	x /= float64(c / 2)
	y /= float64(c / 2)
	p := &input.DispatchMouseEventParams{
		Type:       input.MousePressed,
		X:          x,
		Y:          y,
		Button:     input.Left,
		ClickCount: 1,
	}
	// 鼠标左键按下
	//if err = p.Do(cxt); err != nil {
	//	return err
	//}
	// 拖动
	p.Type = input.MouseMoved
	_max := 380.0
	for {
		if p.X > _max {
			break
		}
		rt := rand.Intn(20) + 20
		_ = chromedp.Run(cxt, chromedp.Sleep(time.Millisecond*time.Duration(rt)))
		_x := rand.Intn(2) + 15
		_y := rand.Intn(2)
		p.X = p.X + float64(_x)
		p.Y = p.Y + float64(_y)
		//fmt.Println("X坐标：",p.X)
		if err = p.Do(cxt); err != nil {
			return err
		}
	}
	// 鼠标松开
	p.Type = input.MouseReleased
	return p.Do(cxt)
}

// 设定一个时间，轮询每个动作，直至超时或者执行成功
func WhileTimeout(timeout time.Duration, roundTimeout time.Duration, returnError bool, actions ...chromedp.Action) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		timer := time.After(timeout)
		for {
			select {
			case <-timer:
				if !returnError {
					return nil
				}
				if err != nil {
					return
				}
				return context.DeadlineExceeded
			default:
				t, cancel := context.WithTimeout(ctx, roundTimeout)
				if err = chromedp.Run(t, actions...); err == nil {
					cancel()
					return nil
				}
				cancel()
				time.Sleep(2 * time.Second)
			}
		}
	}
}

// 设定一个时间，直至超时或者执行成功
func WithTimeout(timeout time.Duration, returnError bool, actions ...chromedp.Action) chromedp.ActionFunc {
	// 执行动作
	return func(ctx context.Context) (err error) {
		t, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err = chromedp.Run(t, actions...); err == nil {
			return nil
		}
		if !returnError {
			return nil
		}
		return
	}
}

func ClickXY(selector string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		_ = clickXY(ctx, selector)
		return
	}
}

func clickXY(ctx context.Context, selector string) (err error) {
	var (
		rect map[string]interface{}
	)

	selector = `{let {x,y} = document.querySelector("` + selector + `").getBoundingClientRect(); let a={x,y}; a;}`
	err = chromedp.Run(ctx, TaskLogger("click xy..."),
		chromedp.Evaluate(selector, &rect))
	if err != nil {
		logger.Sugar().Error(err)
		return
	}

	err = chromedp.Run(ctx, chromedp.MouseClickXY(rect["x"].(float64)+22+12, rect["y"].(float64)+23+12))
	if err != nil {
		logger.Sugar().Error(err)
	}
	return
}

func WaitCLickXY(selector string, timeout time.Duration, actions ...chromedp.Action) chromedp.ActionFunc {
	return WhileTimeout(timeout, 3*time.Second, true, append([]chromedp.Action{ClickXY(selector)}, actions...)...)
}

func Visible(selector string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			obj, exp, err := runtime.Evaluate("document.querySelector('" + selector + "')").Do(ctx)
			if err != nil {
				return err
			}
			if exp != nil {
				return exp
			}

			if obj.ObjectID == "" {
				return errors.New("not visible")
			}
			return nil
		}),
	}
}

func NotVisible(selector string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			obj, exp, err := runtime.Evaluate("document.querySelector('" + selector + "')").Do(ctx)
			if err != nil {
				return err
			}
			if exp != nil {
				return exp
			}

			if obj.ObjectID != "" {
				return errors.New("visible")
			}
			return nil
		}),
	}
}

func NoReturnEvaluate(script string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, exp, err := runtime.Evaluate(script).Do(ctx)
			if err != nil {
				return err
			}
			if exp != nil {
				return exp
			}
			return nil
		}),
	}
}

func EvaluateStealth() chromedp.ActionFunc {
	return EvaluateHookJS(plugins.StealthJs)
}

func EvaluateHook() chromedp.ActionFunc {
	return EvaluateHookJS(plugins.HookJs)
}

func EvaluateHook2() chromedp.ActionFunc {
	return EvaluateHookJS(plugins.HookJs2)
}

func EvaluateTurnstile() chromedp.ActionFunc {
	return EvaluateHookJS(plugins.TurnstileJs)
}

func EvaluateHookJS(hookJS string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		_, err = page.AddScriptToEvaluateOnNewDocument(hookJS).Do(ctx)
		return
	}
}

func EvaluateCallbackManager(events ...string) chromedp.EvaluateAction {
	content := ""
	for _, event := range events {
		content += fmt.Sprintf("\nwindow.CallbackManager.register('%s', function(data) { this.data['%s'] = {ok: true, args: data}; });", event, event)
	}
	return chromedp.Evaluate(`
            // 创建一个回调管理器
            window.CallbackManager = {
                callbacks: {},
				data: {},
                // 注册回调
                register(id, callback) {
                    this.callbacks[id] = callback;
                },
                // 执行回调
                execute(id, data) {
                    if (this.callbacks[id]) {
                        return this.callbacks[id]?.call(this, data);
                    }
                    throw new Error('event "' + id + '" not found');
                },
				callback(id, timeout) {
					return new Promise((resolve, inject) => {
						if (timeout > 0) {
							let inj = false;
							setTimeout(() => { inj = true }, timeout);
							let timer = setInterval(() => {
								if (inj) {
									inject(new Error('timeout'));
									clearInterval(timer);
									return;
								}
								if (window.CallbackManager.data[id]) {
									const data = window.CallbackManager.data[id];
									clearInterval(timer);
									if (data.ok) {
										resolve(data.args);
									} else {
										inject(new Error('not data'));
									}
								}
							}, timeout);
							return;
						}

						if (window.CallbackManager.data[id]?.ok) {
							resolve(window.CallbackManager.data[id]?.args);
							return;
						}
						inject(new Error('not data'));
					});
				}
            };
            // 注册一些示例回调`+content, nil)
}

func EvaluateCallback[T any](name string, timeout time.Duration, result T) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		return chromedp.Evaluate(fmt.Sprintf("window.CallbackManager.callback('%s', %d)", name, timeout), &result, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			if timeout > 0 {
				p = p.WithTimeout(runtime.TimeDelta(timeout))
			}
			return p.WithAwaitPromise(true)
		}).Do(ctx)
	}
}

func EvaluateJS(js string, timeout time.Duration) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		return chromedp.Evaluate(js, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			if timeout > 0 {
				p = p.WithTimeout(runtime.TimeDelta(timeout))
			}
			return p.WithAwaitPromise(true)
		}).Do(ctx)
	}
}

func Assert(condition func(context.Context) bool, messages ...string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		if !condition(ctx) {
			err = errors.New("assert condition is false")
			message := strings.Join(messages, " ")
			if message != "" {
				err = errors.New(message)
			}
		}
		return
	}
}

func Title(ctx context.Context) string {
	t, c := context.WithTimeout(ctx, 3*time.Second)
	defer c()
	var title string
	_ = chromedp.Run(t, chromedp.Title(&title))
	return title
}

func GetCookies(ctx context.Context) (cookies, lang, userAgent string, err error) {
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) (err error) {
		cookieJar, err := network.GetCookies().Do(ctx)
		if err != nil {
			return
		}

		for _, cookie := range cookieJar {
			cookies += cookie.Name + "=" + cookie.Value + "; "
		}

		return
	}),
		chromedp.Evaluate(`navigator.languages.join(',') + ';q=0.9';`, &lang),
		chromedp.Evaluate(`navigator.userAgent;`, &userAgent))
	return
}

func Screenshot(result chan string) chromedp.ActionFunc {
	// 执行动作
	var screenshotBytes []byte
	return func(ctx context.Context) (err error) {
		err = chromedp.Run(ctx, chromedp.CaptureScreenshot(&screenshotBytes))
		if err == nil {
			if !exists("tmp") {
				_ = os.Mkdir("tmp", 0744)
			}

			file := "tmp/screenshot-" + uuid.NewString() + ".png"
			e := os.WriteFile(file, screenshotBytes, 0744)
			if e != nil {
				logger.Sugar().Error("screenshot failed: ", e)
				return
			}

			logger.Sugar().Info("screenshot file: ", file)
			if result != nil {
				result <- file
			}
		}
		return
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
