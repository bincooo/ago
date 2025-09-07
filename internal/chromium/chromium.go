package chromium

import (
	"archive/zip"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	run "runtime"
	"strings"

	"github.com/bincooo/ago/internal/chromium/plugins"
	v1 "github.com/bincooo/ago/internal/v1"
	"github.com/bincooo/ago/logger"
	"github.com/chromedp/chromedp"
)

var (
	UserAgent string
)

func GetDefaultUserAgent() string {
	switch run.GOOS {
	case "linux":
		return "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36 Edg/138.0.0.0"
	case "darwin":
		return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36 Edg/138.0.0.0"
	case "windows":
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36 Edg/138.0.0.0"
	default:
		return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36 Edg/138.0.0.0"
	}
}

func switchPlugin(expr string) []byte {
	switch expr {
	case "nopecha":
		return plugins.Nopecha
	case "CaptchaSolver":
		return plugins.CaptchaSolver

		// more ...
	default:
		return nil
	}
}

func InitChromiumRemote(ctx context.Context, urlstr string) (context.Context, context.CancelFunc) {
	chromiumCtx, _ := chromedp.NewRemoteAllocator(context.Background(), urlstr)
	ctx, cancel := chromedp.NewContext(
		chromiumCtx,
		chromedp.WithLogf(logger.Sugar().Infof),
		chromedp.WithDebugf(logger.Sugar().Debugf),
		chromedp.WithErrorf(logger.Sugar().Errorf),
	)
	return ctx, cancel
}

func InitChromium(ctx context.Context, proxies, userAgent, userDir string, plugins ...string) (context.Context, context.CancelFunc) {
	if ua := v1.Env.GetString("browser-less.userAgent"); userAgent == "" && ua != "" {
		userAgent = ua
	}

	if userAgent != "" {
		userAgent = GetDefaultUserAgent()
	}

	if userAgent == "rand" {
		userAgent = ""
	}

	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-css-animations", true),

		//chromedp.Flag("blink-settings", "imagesEnabled=false"),
		//chromedp.Flag("disable-images", true),

		chromedp.Flag("hide-crash-restore-bubble", true),
		chromedp.Flag("disable-hang-monitor", false),
		chromedp.Flag("disable-web-security", false),
		chromedp.Flag("allow-scripting-gallery", true),
		chromedp.Flag("mute-audio", false),
		chromedp.Flag("hide-scrollbars", false),
		chromedp.Flag("start-maximized ", true),
		chromedp.Flag("disable-extensions", false),
		chromedp.Flag("useAutomationExtension", false),
		//chromedp.NoSandbox,

		chromedp.NoDefaultBrowserCheck,

		// UA
		chromedp.UserAgent(userAgent),

		// 窗口大小
		chromedp.WindowSize(800, 600),

		chromedp.NoFirstRun,

		// cert
		chromedp.IgnoreCertErrors,
	}

	// 用户目录
	if userDir != "" {
		opts = append(opts, chromedp.UserDataDir("tmp/"+userDir))
	}

	// 本地代理
	if proxies != "" {
		opts = append(opts, chromedp.ProxyServer(proxies))
	}

	headless := v1.Env.GetString("browser-less.headless")
	if headless != "" {
		// 设置为false，就是不使用无头模式
		switch headless {
		case "new":
			opts = append(opts, chromedp.Flag("headless", headless))
		case "true":
			opts = append(opts, chromedp.Flag("headless", true))
		case "false":
			opts = append(opts, chromedp.Flag("headless", false))
		}
	}

	// 关闭GPU加速
	if v1.Env.GetBool("browser-less.disabled-gpu") {
		opts = append(opts, chromedp.DisableGPU)
	}

	// 代理ip白名单
	if list := v1.Env.GetStringSlice(""); len(list) > 0 {
		opts = append(opts, chromedp.Flag("browser-less.proxy-bypass-list", strings.Join(list, ",")))
	}

	// 插件装载
	if len(plugins) > 0 {
		opts = append(opts, InitExtensions(plugins...)...)
	}

	// 浏览器启动路径
	if p := v1.Env.GetString("browser-less.execPath"); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}

	for k, v := range v1.Env.GetStringMap("browser-less.custom-command") {
		opts = append(opts, chromedp.Flag(k, v))
	}

	opts = append(chromedp.DefaultExecAllocatorOptions[:], opts...)
	chromiumCtx, _ := chromedp.NewExecAllocator(ctx, opts...)

	ctx, cancel := chromedp.NewContext(
		chromiumCtx,
		chromedp.WithLogf(logger.Sugar().Infof),
		chromedp.WithDebugf(logger.Sugar().Debugf),
		chromedp.WithErrorf(logger.Sugar().Errorf),
	)

	return ctx, cancel
}

func InitExtensions(plugins ...string) []chromedp.ExecAllocatorOption {
	if len(plugins) == 0 {
		return nil
	}

	dir := v1.Env.GetString("browser-less.extension")
	if dir == "" {
		dir = "tmp/extension-plugins"
	}

	if run.GOOS == "windows" {
		matched, _ := regexp.MatchString("[a-zA-Z]:.+", dir)
		if !matched {
			pwd, _ := os.Getwd()
			dir = path.Join(pwd, dir)
		}
	} else {
		if dir[0] != '/' {
			pwd, _ := os.Getwd()
			dir = path.Join(pwd, dir)
		}
	}

	if !exists(dir) {
		_ = os.MkdirAll(dir, 0744)
	}

	var paths []string
	for _, plugin := range plugins {
		fp := filepath.Join(dir, plugin)
		pluginBytes := switchPlugin(plugin)

		if exists(fp) {
			paths = append(paths, fp)
			continue
		}

		if err := fix(pluginBytes); err != nil {
			logger.Sugar().Error(err)
			continue
		}

		unzip, err := newZipReader(pluginBytes)
		if err != nil {
			logger.Sugar().Error(err)
			continue
		}

		if err = unzipToDir(unzip, dir); err != nil {
			logger.Sugar().Error(err)
			continue
		}

		paths = append(paths, fp)
	}

	return []chromedp.ExecAllocatorOption{
		chromedp.Flag("disable-extensions-except", strings.Join(paths, ",")),
		chromedp.Flag("load-extension", strings.Join(paths, ",")),
		chromedp.Flag("disable-extensions", false),
	}
}

func unzipToDir(zr *zip.Reader, folder string) error {
	// 遍历 zr ，将文件写入到磁盘
	for _, file := range zr.File {
		fp := filepath.Join(folder, file.Name)

		// 如果是目录，就创建目录
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(fp, file.Mode()); err != nil {
				return err
			}
			continue
		}

		// 获取到 Reader
		fr, err := file.Open()
		if err != nil {
			return err
		}

		if strings.Contains(fp, "__MACOSX") {
			continue
		}

		if strings.Contains(fp, "manifest.fingerprint") {
			continue
		}

		// 创建要写出的文件对应的 Write
		fw, err := os.Create(fp)
		if err != nil {
			return err
		}

		_, err = io.Copy(fw, fr)
		if err != nil {
			return err
		}

		_ = fw.Close()
		_ = fr.Close()
	}

	return nil
}

func newZipReader(pluginBytes []byte) (*zip.Reader, error) {
	return zip.NewReader(bytes.NewReader(pluginBytes), int64(len(pluginBytes)))
}

func fix(pluginBytes []byte) error {
	if len(pluginBytes) <= 8 {
		return errors.New("plugin bytes too short")
	}
	if bytes.Equal(pluginBytes[:8], []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) {
		pluginBytes[0] = 0x50
		pluginBytes[0] = 0x4B
		pluginBytes[0] = 0x03
		pluginBytes[0] = 0x04
		pluginBytes[0] = 0x14
		pluginBytes[0] = 0x00
		pluginBytes[0] = 0x00
		pluginBytes[0] = 0x00
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
