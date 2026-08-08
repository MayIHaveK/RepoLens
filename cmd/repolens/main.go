package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/MayIHaveK/RepoLens/internal/analysis"
	"github.com/MayIHaveK/RepoLens/internal/config"
	"github.com/MayIHaveK/RepoLens/internal/exporthtml"
	"github.com/MayIHaveK/RepoLens/internal/model"
	"github.com/MayIHaveK/RepoLens/internal/server"
	"github.com/MayIHaveK/RepoLens/internal/storage"
)

var version = "0.1.0-dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "repolens:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return serve(nil)
	}
	switch arguments[0] {
	case "serve":
		return serve(arguments[1:])
	case "analyze":
		return analyze(arguments[1:])
	case "export":
		return export(arguments[1:])
	case "version", "--version", "-v":
		fmt.Println("RepoLens", version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("未知命令 %q", arguments[0])
	}
}

func serve(arguments []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	address := flags.String("address", "127.0.0.1:41739", "HTTP 监听地址")
	noOpen := flags.Bool("no-open", false, "启动后不自动打开浏览器")
	cacheDirectory := flags.String("cache", "", "分析缓存目录")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if !isLoopbackAddress(*address) {
		return errors.New("为保护本地仓库，拒绝监听非回环地址；远程访问请使用带身份验证的反向代理")
	}
	store, err := storage.New(*cacheDirectory)
	if err != nil {
		return err
	}
	application := server.New(store, slog.Default())
	httpServer := &http.Server{
		Addr: *address, Handler: application.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 0, IdleTimeout: 60 * time.Second,
	}
	url := "http://" + *address
	fmt.Println("RepoLens 已启动：", url)
	if !*noOpen {
		go func() {
			time.Sleep(350 * time.Millisecond)
			_ = openBrowser(url)
		}()
	}
	return httpServer.ListenAndServe()
}

func analyze(arguments []string) error {
	flags := flag.NewFlagSet("analyze", flag.ContinueOnError)
	ref := flags.String("ref", "HEAD", "需要分析的 Git 版本")
	output := flags.String("output", "repolens-analysis.json", "输出的 JSON 文件")
	profile := flags.String("profile", "balanced", "性能配置：fast、balanced 或 thorough")
	repository := ""
	if len(arguments) > 0 && !strings.HasPrefix(arguments[0], "-") {
		repository = arguments[0]
		arguments = arguments[1:]
	}
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if repository == "" {
		if flags.NArg() == 1 {
			repository = flags.Arg(0)
		} else {
			return errors.New("用法：repolens analyze <仓库路径>")
		}
	} else if flags.NArg() != 0 {
		return errors.New("用法：repolens analyze <仓库路径>")
	}
	cfg := config.Default()
	cfg.Ref = *ref
	cfg.Profile = *profile
	result, err := analysis.New().Analyze(context.Background(), repository, cfg, func(progress model.Progress) {
		fmt.Printf("\r%-12s %5.1f%%  %s", progress.Phase, progress.Percent, progress.Message)
	})
	fmt.Println()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*output, data, 0o600); err != nil {
		return err
	}
	fmt.Println("分析结果已写入：", *output)
	return nil
}

func export(arguments []string) error {
	if len(arguments) != 2 {
		return errors.New("用法：repolens export <分析结果.json> <报告.html>")
	}
	data, err := os.ReadFile(arguments[0])
	if err != nil {
		return err
	}
	var result model.Analysis
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	document, err := exporthtml.Render(&result, result.Config.Privacy)
	if err != nil {
		return err
	}
	if err := os.WriteFile(arguments[1], document, 0o600); err != nil {
		return err
	}
	fmt.Println("报告已写入：", arguments[1])
	return nil
}

func printUsage() {
	fmt.Println(`RepoLens - 本地优先的 Git 贡献分析工具

用法：
  repolens serve [选项]
  repolens analyze <仓库路径> [选项]
  repolens export <analysis.json> <report.html>
  repolens version`)
}

func isLoopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

func openBrowser(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	return command.Start()
}
