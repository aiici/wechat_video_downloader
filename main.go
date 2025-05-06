package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/qtgolang/SunnyNet/SunnyNet"
	"github.com/qtgolang/SunnyNet/public"

	"wechat_video_downloader/pkg/argv"
	"wechat_video_downloader/pkg/certificate"
	"wechat_video_downloader/pkg/proxy"
	"wechat_video_downloader/pkg/util"
)

//go:embed certs/SunnyRoot.cer
var cert_data []byte

//go:embed lib/FileSaver.min.js
var file_saver_js []byte

//go:embed lib/jszip.min.js
var zip_js []byte

//go:embed inject/main.js
var main_js []byte

var Sunny = SunnyNet.NewSunny()
var version = "2025050601"
var v = "?t=" + version
var port = 2025

// 打印帮助信息
func print_usage() {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println(bold("微信视频号下载器 (WeChat Video Downloader)"))
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("%s\n\n", cyan("一个用于下载微信视频号视频的专业工具。\n"))

	fmt.Println(bold("用法 (Usage):"))
	fmt.Printf("  %s %s\n\n", green("wechat_video_downloader"), cyan("[OPTIONS]"))

	fmt.Println(bold("可用选项 (Options):"))
	fmt.Printf("  %s, %s\t%s\n", green("-h"), green("--help"), "显示帮助信息并退出 (Show this help and exit)")
	fmt.Printf("  %s, %s\t%s\n", green("-v"), green("--version"), "显示版本信息并退出 (Show version information and exit)")
	fmt.Printf("  %s, %s\t%s\n", green("-p"), green("--port"), "设置代理服务器端口 (Set proxy server network port)")
	fmt.Printf("  %s, %s\t%s\n", green("-d"), green("--dev"), "设置网络设备 (Set proxy server network device)")

	fmt.Println("\n示例 (Examples):")
	fmt.Printf("  %s\n", cyan("wechat_video_downloader -p 2025"))
	fmt.Printf("  %s\n", cyan("wechat_video_downloader --dev Ethernet0"))

	fmt.Println("\n更多信息请访问：https://github.com/aiici/wechat_video_downloader")
}

// 处理命令行参数
func parseArgs() (map[string]string, string) {
	os_env := runtime.GOOS
	args := argv.ArgsToMap(os.Args) // 分解参数列表为Map
	
	// 处理帮助和版本信息
	if _, ok := args["help"]; ok {
		print_usage()
		os.Exit(0)
	}
	if v, ok := args["v"]; ok || args["version"] != "" {
		fmt.Printf("v%s %.0s\n", version, v)
		os.Exit(0)
	}
	
	// 设置参数默认值
	args["dev"] = argv.ArgsValue(args, "", "d", "dev")
	args["port"] = argv.ArgsValue(args, "", "p", "port")
	iport, errstr := strconv.Atoi(args["port"])
	if errstr != nil {
		args["port"] = strconv.Itoa(port) // 用户自定义值解析失败则使用默认端口
	} else {
		port = iport
	}

	delete(args, "p") // 删除冗余的参数p
	delete(args, "d") // 删除冗余的参数d
	
	return args, os_env
}

// 设置信号处理
func setupSignalHandler(args map[string]string, os_env string) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalChan
		logMessage(LogLevelInfo,"\n正在关闭服务并退出...(%v)", sig)
		if os_env == "darwin" {
			proxy.DisableProxyInMacOS(proxy.ProxySettings{
				Device:   args["dev"],
				Hostname: "127.0.0.1",
				Port:     args["port"],
			})
		}
		os.Exit(0)
	}()
}

// 检查并安装证书
func checkAndInstallCertificate() error {
	existing, err := certificate.CheckCertificate("SunnyNet")
	if err != nil {
		return fmt.Errorf("检查证书失败: %v", err)
	}
	
	if !existing {
		fmt.Printf("\n正在安装证书...\n")
		err := certificate.InstallCertificate(cert_data)
		time.Sleep(3 * time.Second)
		if err != nil {
			return fmt.Errorf("安装证书失败: %v", err)
		}
	}
	
	return nil
}

// 启动代理服务
func startProxyServer() error {
	Sunny.SetPort(port)
	Sunny.SetGoCallback(HttpCallback, nil, nil, nil)
	err := Sunny.Start().Error
	if err != nil {
		return fmt.Errorf("启动代理服务失败: %v", err)
	}
	
	return nil
}

// 配置系统代理
func configureSystemProxy(args map[string]string, os_env string) error {
	proxy_server := fmt.Sprintf("127.0.0.1:%v", port)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "http",
				Host:   proxy_server,
			}),
		},
	}
	
	_, err := client.Get("https://sunny.io/")
	if err != nil {
		logMessage(LogLevelInfo,fmt.Sprintf("\n您还未安装证书，请在浏览器打开 http://%v 并根据说明安装证书\n在安装完成后重新启动此程序即可\n", proxy_server))
		return nil
	}
	
	if os_env != "windows" {
		return fmt.Errorf("抱歉，暂不支持此操作系统")
	}
	ok := Sunny.StartProcess()
	if !ok {
		return fmt.Errorf("启动进程代理失败，检查是否以管理员身份运行")
	}
	Sunny.ProcessAddName("WeChatAppEx.exe")

	
	
	// if os_env == "darwin" {
	// 	err := proxy.EnableProxyInMacOS(proxy.ProxySettings{
	// 		Device:   args["dev"],
	// 		Hostname: "127.0.0.1",
	// 		Port:     args["port"],
	// 	})
	// 	if err != nil {
	// 		return fmt.Errorf("设置代理失败: %v", err)
	// 	}
	// }
	
	color.Green(fmt.Sprintf("\n服务已正确启动，请打开需要下载的视频号页面进行下载\n\n"))
	return nil
}

// 处理错误并等待用户退出
func handleError(err error) {
	logMessage(LogLevelError,"\nERROR %v\n", err.Error())
	logMessage(LogLevelInfo,"按 Ctrl+C 退出...\n")
	select {}
}

// 日志级别
const (
	LogLevelInfo  = "INFO"
	LogLevelError = "ERROR"
	LogLevelDebug = "DEBUG"
)

// 打印日志
func logMessage(level string, format string, args ...interface{}) {
	var prefix string
	switch level {
	case LogLevelInfo:
		prefix = color.GreenString("[INFO] ")
	case LogLevelError:
		prefix = color.RedString("[ERROR] ")
	case LogLevelDebug:
		prefix = color.YellowString("[DEBUG] ")
	default:
		prefix = color.WhiteString("[LOG] ")
	}
	
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s\n", prefix, message)
}

type ChannelProfile struct {
	Title string `json:"title"`
}
type FrontendTip struct {
	Msg string `json:"msg"`
}

func HttpCallback(Conn *SunnyNet.HttpConn) {
	host := Conn.Request.URL.Hostname()
	path := Conn.Request.URL.Path
	if Conn.Type == public.HttpSendRequest {
		// Conn.Request.Header.Set("Cache-Control", "no-cache")
		Conn.Request.Header.Del("Accept-Encoding")
		if util.Includes(path, "jszip") {
			headers := http.Header{}
			headers.Set("Content-Type", "application/javascript")
			headers.Set("__debug", "local_file")
			Conn.StopRequest(200, zip_js, headers)
			return
		}
		if util.Includes(path, "FileSaver.min") {
			headers := http.Header{}
			headers.Set("Content-Type", "application/javascript")
			headers.Set("__debug", "local_file")
			Conn.StopRequest(200, file_saver_js, headers)
			return
		}
		if path == "/__wx_channels_api/profile" {
			var data ChannelProfile
			body, _ := io.ReadAll(Conn.Request.Body)
			_ = Conn.Request.Body.Close()
			err := json.Unmarshal(body, &data)
			if err != nil {
				fmt.Println(err.Error())
			}
			logMessage(LogLevelDebug,"\n打开了视频\n%s\n", data.Title)
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("__debug", "fake_resp")
			Conn.StopRequest(200, "{}", headers)
			return
		}
		if path == "/__wx_channels_api/tip" {
			var data FrontendTip
			body, _ := io.ReadAll(Conn.Request.Body)
			_ = Conn.Request.Body.Close()
			err := json.Unmarshal(body, &data)
			if err != nil {
				fmt.Println(err.Error())
			}
			
			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			headers.Set("__debug", "fake_resp")
			Conn.StopRequest(200, "{}", headers)
			return
		}
	}
	if Conn.Type == public.HttpResponseOK {
		content_type := strings.ToLower(Conn.Response.Header.Get("content-type"))
		if Conn.Response.Body != nil {
			Body, _ := io.ReadAll(Conn.Response.Body)
			_ = Conn.Response.Body.Close()

			if content_type == "text/html; charset=utf-8" {
				html := string(Body)
				script_reg1 := regexp.MustCompile(`src="([^"]{1,})\.js"`)
				html = script_reg1.ReplaceAllString(html, `src="$1.js`+v+`"`)
				script_reg2 := regexp.MustCompile(`href="([^"]{1,})\.js"`)
				html = script_reg2.ReplaceAllString(html, `href="$1.js`+v+`"`)
				Conn.Response.Header.Set("__debug", "append_script")
				script2 := ""
				if host == "channels.weixin.qq.com" && (path == "/web/pages/feed" || path == "/web/pages/home") {
					// Conn.Response.Header.Add("wx-channel-video-download", "1")
					script := fmt.Sprintf(`<script>%s</script>`, main_js)
					html = strings.Replace(html, "<head>", "<head>\n"+script+script2, 1)
					logMessage(LogLevelInfo,"1. 视频详情页 html 注入 js 成功")
					Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(html)))
					return
				}
				Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(html)))
				return
			}
			if content_type == "application/javascript" {
				content := string(Body)
				dep_reg := regexp.MustCompile(`"js/([^"]{1,})\.js"`)
				from_reg := regexp.MustCompile(`from {0,1}"([^"]{1,})\.js"`)
				lazy_import_reg := regexp.MustCompile(`import\("([^"]{1,})\.js"\)`)
				import_reg := regexp.MustCompile(`import {0,1}"([^"]{1,})\.js"`)
				content = from_reg.ReplaceAllString(content, `from"$1.js`+v+`"`)
				content = dep_reg.ReplaceAllString(content, `"js/$1.js`+v+`"`)
				content = lazy_import_reg.ReplaceAllString(content, `import("$1.js`+v+`")`)
				content = import_reg.ReplaceAllString(content, `import"$1.js`+v+`"`)
				Conn.Response.Header.Set("__debug", "replace_script")

				if util.Includes(path, "/t/wx_fed/finder/web/web-finder/res/js/index.publish") {
					regexp1 := regexp.MustCompile(`this.sourceBuffer.appendBuffer\(h\),`)
					replaceStr1 := `(() => {
if (window.__wx_channels_store__) {
window.__wx_channels_store__.buffers.push(h);
}
})(),this.sourceBuffer.appendBuffer(h),`
					if regexp1.MatchString(content) {
						logMessage(LogLevelInfo,"2. 视频播放 js 修改成功")
					}
					content = regexp1.ReplaceAllString(content, replaceStr1)
					regexp2 := regexp.MustCompile(`if\(f.cmd===re.MAIN_THREAD_CMD.AUTO_CUT`)
					replaceStr2 := `if(f.cmd==="CUT"){
	if (window.__wx_channels_store__) {
	console.log("CUT", f, __wx_channels_store__.profile.key);
	window.__wx_channels_store__.keys[__wx_channels_store__.profile.key]=f.decryptor_array;
	}
}
if(f.cmd===re.MAIN_THREAD_CMD.AUTO_CUT`
					content = regexp2.ReplaceAllString(content, replaceStr2)
					Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
					return
				}
				if util.Includes(path, "/t/wx_fed/finder/web/web-finder/res/js/virtual_svg-icons-register") {
					regexp1 := regexp.MustCompile(`async finderGetCommentDetail\((\w+)\)\{return(.*?)\}async`)
					replaceStr1 := `async finderGetCommentDetail($1) {
					var feedResult = await$2;
					var data_object = feedResult.data.object;
					if (!data_object.objectDesc) {
						return feedResult;
					}
					var media = data_object.objectDesc.media[0];
					var profile = media.mediaType !== 4 ? {
						type: "picture",
						id: data_object.id,
						title: data_object.objectDesc.description,
						files: data_object.objectDesc.media,
						spec: [],
						contact: data_object.contact
					} : {
						type: "media",
						duration: media.spec[0].durationMs,
						spec: media.spec,
						title: data_object.objectDesc.description,
						coverUrl: media.coverUrl,
						url: media.url+media.urlToken,
						size: media.fileSize,
						key: media.decodeKey,
						id: data_object.id,
						nonce_id: data_object.objectNonceId,
						nickname: data_object.nickname,
						createtime: data_object.createtime,
						fileFormat: media.spec.map(o => o.fileFormat),
						contact: data_object.contact
					};
					fetch("/__wx_channels_api/profile", {
						method: "POST",
						headers: {
							"Content-Type": "application/json"
						},
						body: JSON.stringify(profile)
					});
					if (window.__wx_channels_store__) {
					__wx_channels_store__.profile = profile;
					window.__wx_channels_store__.profiles.push(profile);
					}
					return feedResult;
				}async`
					if regexp1.MatchString(content) {
						logMessage(LogLevelInfo,"3. 视频详情页 js 修改成功")
					}
					content = regexp1.ReplaceAllString(content, replaceStr1)
					regex2 := regexp.MustCompile(`i.default={dialog`)
					replaceStr2 := `i.default=window.window.__wx_channels_tip__={dialog`
					content = regex2.ReplaceAllString(content, replaceStr2)
					regex5 := regexp.MustCompile(`this.updateDetail\(o\)`)
					replaceStr5 := `(() => {
					if (Object.keys(o).length===0){
					return;
					}
					var data_object = o;
					var media = data_object.objectDesc.media[0];
					var profile = media.mediaType !== 4 ? {
						type: "picture",
						id: data_object.id,
						title: data_object.objectDesc.description,
						files: data_object.objectDesc.media,
						spec: [],
						contact: data_object.contact
					} : {
						type: "media",
						duration: media.spec[0].durationMs,
						spec: media.spec,
						title: data_object.objectDesc.description,
						url: media.url+media.urlToken,
						size: media.fileSize,
						key: media.decodeKey,
						id: data_object.id,
						nonce_id: data_object.objectNonceId,
						nickname: data_object.nickname,
						createtime: data_object.createtime,
						fileFormat: media.spec.map(o => o.fileFormat),
						contact: data_object.contact
					};
					if (window.__wx_channels_store__) {
window.__wx_channels_store__.profiles.push(profile);
					}
					})(),this.updateDetail(o)`
					content = regex5.ReplaceAllString(content, replaceStr5)
					Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
					return
				}
				if util.Includes(path, "/t/wx_fed/finder/web/web-finder/res/js/FeedDetail.publish") {
					regex := regexp.MustCompile(`,"投诉"\)]`)
					replaceStr := `,"真投诉"),...(() => {
					if (window.__wx_channels_store__ && window.__wx_channels_store__.profile) {
						return window.__wx_channels_store__.profile.spec.map((sp) => {
							return f("div",{class:"context-item",role:"button",onClick:() => __wx_channels_handle_click_download__(sp)},sp.fileFormat);
						});
					}
					})(),f("div",{class:"context-item",role:"button",onClick:()=>__wx_channels_handle_click_download__()},"原始视频"),f("div",{class:"context-item",role:"button",onClick:__wx_channels_download_cur__},"当前视频"),f("div",{class:"context-item",role:"button",onClick:()=>__wx_channels_handle_download_cover()},"下载封面"),f("div",{class:"context-item",role:"button",onClick:__wx_channels_handle_copy__},"复制页面链接")]`
					content = regex.ReplaceAllString(content, replaceStr)
					Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
					return
				}
				if util.Includes(path, "worker_release") {
					regex := regexp.MustCompile(`fmp4Index:p.fmp4Index`)
					replaceStr := `decryptor_array:p.decryptor_array,fmp4Index:p.fmp4Index`
					content = regex.ReplaceAllString(content, replaceStr)
					Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
					return
				}
				Conn.Response.Body = io.NopCloser(bytes.NewBuffer([]byte(content)))
				return
			}
			Conn.Response.Body = io.NopCloser(bytes.NewBuffer(Body))
		}

	}
	if Conn.Type == public.HttpRequestFail {
		logMessage(LogLevelDebug,"%s %s", Conn.Request.Method, Conn.Request.URL)
	}
}

func main() {
	args, os_env := parseArgs()
	setupSignalHandler(args, os_env)
	
	print_usage()
	
	// 检查并安装证书
	if err := checkAndInstallCertificate(); err != nil {
		logMessage(LogLevelError, "%v", err)
		handleError(err)
	}
	
	// 启动代理服务
	if err := startProxyServer(); err != nil {
		logMessage(LogLevelError, "%v", err)
		handleError(err)
	}
	
	// 配置系统代理
	if err := configureSystemProxy(args, os_env); err != nil {
		logMessage(LogLevelError, "%v", err)
		handleError(err)
	}
	
	logMessage(LogLevelInfo, "服务正在运行，按 Ctrl+C 退出...")
	select {}
}