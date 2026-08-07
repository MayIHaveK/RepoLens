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
	"time"

	"github.com/repolens/repolens/internal/analysis"
	"github.com/repolens/repolens/internal/config"
	"github.com/repolens/repolens/internal/exporthtml"
	"github.com/repolens/repolens/internal/model"
	"github.com/repolens/repolens/internal/server"
	"github.com/repolens/repolens/internal/storage"
)

const version = "0.1.0-dev"

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
		return fmt.Errorf("unknown command %q", arguments[0])
	}
}

func serve(arguments []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	address := flags.String("address", "127.0.0.1:41739", "HTTP listen address")
	noOpen := flags.Bool("no-open", false, "do not open the browser")
	cacheDirectory := flags.String("cache", "", "analysis cache directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if !isLoopbackAddress(*address) {
		return errors.New("refusing to bind outside loopback; use a reverse proxy with authentication for remote access")
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
	fmt.Println("RepoLens is running at", url)
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
	ref := flags.String("ref", "HEAD", "Git ref to analyze")
	output := flags.String("output", "repolens-analysis.json", "output JSON file")
	profile := flags.String("profile", "balanced", "fast, balanced, or thorough")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: repolens analyze <repository>")
	}
	cfg := config.Default()
	cfg.Ref = *ref
	cfg.Profile = *profile
	result, err := analysis.New().Analyze(context.Background(), flags.Arg(0), cfg, func(progress model.Progress) {
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
	fmt.Println("Analysis written to", *output)
	return nil
}

func export(arguments []string) error {
	if len(arguments) != 2 {
		return errors.New("usage: repolens export <analysis.json> <report.html>")
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
	fmt.Println("Report written to", arguments[1])
	return nil
}

func printUsage() {
	fmt.Println(`RepoLens - local-first Git contribution analytics

Usage:
  repolens serve [options]
  repolens analyze <repository> [options]
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
