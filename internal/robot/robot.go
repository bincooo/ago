package robot

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	v1 "github.com/bincooo/ago/internal/v1"
	"github.com/elazarl/goproxy"
	"github.com/go-vgo/robotgo"
	"github.com/robotn/gohook"

	_ "embed"
)

var (
	chanToken = make(chan string)

	//go:embed ca.pem
	ca []byte
	//go:embed key.pem
	key []byte

	mu sync.Mutex

	TimeoutErr = errors.New("context timeout")
)

func WaitEvent(ctx context.Context) (token string, err error) {
	timeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	mu.Lock()
	defer mu.Unlock()

	runAuto(timeout)
	if isCancel(timeout) {
		return
	}

	timeout, cancel = context.WithTimeout(timeout, 3*time.Second)
	defer cancel()

	select {
	case <-timeout.Done():
		err = errors.New("context timeout")
		return
	case token = <-chanToken:
		return
	}

	// TODO -
}

func Hook() {
	eventHook()
}

func Run(proxies string) {
	proxy := goproxy.NewProxyHttpServer()
	setCert(ca, key)

	proxy.Verbose = false
	proxy.AllowHTTP2 = true
	proxy.KeepHeader = true
	proxy.KeepDestinationHeaders = true
	if proxies != "" {
		proxy.Tr.TLSClientConfig.InsecureSkipVerify = true
		proxy.Tr.Proxy = func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxies)
		}
	}

	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		str := ctx.Req.URL.String()
		log.Println("OnRequest:", ctx.Req.Proto, ctx.Req.Method, str)

		// TODO -

		slice := v1.Env.GetStringSlice("hook")
		for _, content := range slice {
			split := strings.Split(content, "|")
			if len(split) < 3 {
				continue
			}
			condition := (split[1] == "f" && str == split[2]) ||
				(split[1] == "le" && strings.HasPrefix(str, split[2])) ||
				(split[1] == "re" && strings.HasSuffix(str, split[2])) ||
				(split[1] == "c" && strings.Contains(str, split[2]))
			if condition {
				chunk, err := io.ReadAll(req.Body)
				if err != nil {
					panic(err)
				}
				if split[0] == "hex" {
					println(hex.EncodeToString(chunk))
				} else {
					println(string(chunk))
				}
				break
			}
		}

		return req, ctx.Resp
	})

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		log.Println("OnResponse:", ctx.Req.Proto, ctx.Req.Method, ctx.Req.URL.String())
		return resp
	})

	go func() { log.Fatal(http.ListenAndServe(":17890", proxy)) }()
}

func eventHook() {
	pid, err := runApp(nil)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err = robotgo.Kill(pid)
		if err != nil {
			log.Fatal(err)
		}
	}()

	hook.Register(hook.MouseDown, []string{}, func(ev hook.Event) {
		if ev.Button == hook.MouseMap["left"] {
			fmt.Printf("mouse left @ x = %d - y = %d\n", ev.X, ev.Y)
		}
	})

	s := hook.Start()
	hook.Register(hook.KeyDown, []string{"esc"}, func(ev hook.Event) {
		close(s)
	})
	<-hook.Process(s)
}

func setCert(caCert, caKey []byte) {
	gCa, _ := tls.X509KeyPair(caCert, caKey)
	gCa.Leaf, _ = x509.ParseCertificate(gCa.Certificate[0])
	goproxy.GoproxyCa = gCa
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: goproxy.TLSConfigFromCA(&gCa)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&gCa)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: goproxy.TLSConfigFromCA(&gCa)}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: goproxy.TLSConfigFromCA(&gCa)}
}

func runAuto(ctx context.Context) {
	pid, err := runApp(ctx)
	if err != nil {
		log.Println(err)
		return
	}

	waitMillisecond := v1.Env.GetInt64("app.waitMillisecond")
	sW, sH := robotgo.GetScreenSize()
	var events []map[string]interface{}
	err = v1.Env.UnmarshalKey("app.events", &events)
	if err != nil {
		log.Println(err)
	}

	x := 0
	y := 0

	ax := v1.Env.GetInt("app.x")
	ay := v1.Env.GetInt("app.y")
	if ax != 0 && ay != 0 {
		x = sW/2 + ax
		y = sH/2 + ay
	}

	if isCancel(ctx) {
		return
	}

	for _, event := range events {
		if isCancel(ctx) {
			return
		}

		for k, v := range event {
			switch k {
			case "sleep":
				time.Sleep(time.Duration(v.(int)) * time.Millisecond)
				break
			case "move":
				obj := v.(map[string]interface{})
				x1 := obj["x"].(int)
				y1 := obj["y"].(int)
				robotgo.Move(x+x1, y+y1, pid)
			case "click":
				obj := v.(map[string]interface{})
				k1 := obj["key"].(string)
				double := obj["double"].(bool)
				robotgo.Click(k1, double)
			case "tap":
				obj := v.(map[string]interface{})
				k1 := obj["keys"].([]interface{})
				err = robotgo.KeyTap(k1[0].(string), k1[1:]...)
				println(err)
			case "send":
				robotgo.TypeStr(v.(string))
			}
			time.Sleep(time.Duration(waitMillisecond) * time.Millisecond)
			break
		}
	}

	time.Sleep(time.Second)
	err = robotgo.Kill(pid)
	//err = exec.Command("kill", "-9", fmt.Sprintf("%d", pid)).Run()
	if err != nil {
		log.Println(err)
	}
}

func runApp(ctx context.Context) (pid int, err error) {
	fpid, err := robotgo.FindIds(v1.Env.GetString("app.name"))
	if err != nil {
		log.Println(err)
	}

	if len(fpid) == 0 {
		go func() {
			_, err = robotgo.Run(v1.Env.GetString("app.path"))
			if err != nil {
				log.Printf("starting err: %v", err)
			}
		}()

		time.Sleep(time.Second)
		fpid, err = robotgo.FindIds(v1.Env.GetString("app.name"))
	}

	if isCancel(ctx) {
		return -1, TimeoutErr
	}

	if len(fpid) == 0 {
		return -1, errors.New("not pid")
	}

	pid = fpid[0]
	err = robotgo.ActivePid(pid)
	if err != nil {
		return -1, err
	}

	return
}

func isCancel(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
