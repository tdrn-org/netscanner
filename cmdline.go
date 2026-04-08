/*
 * Copyright 2025-2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package netscanner

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/tdrn-org/go-diff"
	"github.com/tdrn-org/netscanner/internal/buildinfo"
)

var cmdLineApplication = kong.Name(buildinfo.Cmd())

var cmdLineHelpOptions = kong.ConfigureHelp(kong.HelpOptions{
	Compact: true,
})

var cmdLineVars = kong.Vars{
	"config_default": DefaultConfigPath(),
}

type cmdLine struct {
	Silent      bool            `short:"s" help:"Enable silent mode (log level error)"`
	Quiet       bool            `short:"q" help:"Enable quiet mode (log level warn)"`
	Verbose     bool            `short:"v" help:"Enable verbose output (log level info)"`
	Debug       bool            `short:"d" help:"Enable debug output (log level debug)"`
	RunCmd      runCmd          `cmd:"" name:"run" default:"withargs" help:"Run server"`
	VersionCmd  versionCmd      `cmd:"" name:"version" help:"Show version info"`
	TemplateCmd templateCmd     `cmd:"" name:"template" help:"Output config template"`
	ctx         context.Context `kong:"-"`
}

type runCmd struct {
	Config string `short:"c" help:"The configuration file to use" default:"${config_default}"`
}

func (cmd *runCmd) Run(args *cmdLine) error {
	path := strings.TrimSpace(cmd.Config)
	if path == "" {
		path = DefaultConfigPath()
	}
	config, err := LoadConfig(path, false)
	if err != nil {
		return err
	}
	cmd.applyGlobalArgs(config, args)
	config.Logging.apply()
	server, err := StartServer(args.ctx, config)
	if err != nil {
		return err
	}
	stoppedWG := sync.WaitGroup{}
	stoppedWG.Go(func() {
		err = server.Run(args.ctx)
	})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		sigintCtx, cancelListenAndServe := context.WithCancel(args.ctx)
		go func() {
			<-sigint
			slog.Info("signal SIGINT; stopping")
			cancelListenAndServe()
		}()
		<-sigintCtx.Done()
		server.Shutdown(args.ctx)
	}()
	stoppedWG.Wait()
	if err == nil {
		slog.Info("stopped")
	}
	return errors.Join(err, server.Close())
}

func (cmd *runCmd) applyGlobalArgs(config *Config, args *cmdLine) {
	if args.Debug {
		config.Logging.Level = LogLevel(slog.LevelDebug)
	} else if args.Verbose {
		config.Logging.Level = LogLevel(slog.LevelInfo)
	} else if args.Quiet {
		config.Logging.Level = LogLevel(slog.LevelWarn)
	} else if args.Silent {
		config.Logging.Level = LogLevel(slog.LevelError)
	}
}

type versionCmd struct {
	Extended bool `short:"x" help:"Output extended version info"`
}

func (cmd *versionCmd) Run(args *cmdLine) error {
	fmt.Println(buildinfo.FullVersion())
	if args.VersionCmd.Extended {
		fmt.Println(buildinfo.Extended())
	}
	return nil
}

type templateCmd struct {
	Diff    string `help:"The configuration file to compare the config template to"`
	Unified bool   `short:"u" help:"Print diff in unified format"`
	NoAnsi  bool   `help:"Disable colored output"`
	Ansi    bool   `help:"Force colored output"`
}

//go:embed config_template.toml
var configTemplate string

func (cmd *templateCmd) Run(args *cmdLine) error {
	if cmd.Diff == "" {
		fmt.Print(configTemplate)
	} else {
		diffFile, err := os.Open(cmd.Diff)
		if err != nil {
			return fmt.Errorf("unable to open file '%s' (cause: %w)", cmd.Diff, err)
		}
		defer diffFile.Close()
		diffResult, err := diff.Diff(strings.NewReader(configTemplate), diffFile)
		if err != nil {
			return fmt.Errorf("failed to compare configurations (cause: %w)", err)
		}
		diffResult.LeftName = "idpd.toml"
		diffResult.RightName = diffFile.Name()
		printerOptions := make([]diff.PrinterOption, 0, 2)
		if cmd.NoAnsi {
			printerOptions = append(printerOptions, diff.WithAnsi(false))
		} else if cmd.Ansi {
			printerOptions = append(printerOptions, diff.WithAnsi(true))
		}
		if cmd.Unified {
			printerOptions = append(printerOptions, diff.WithUnifiedFormatter(diff.DefaultUnifiedContext))
		}
		diff.NewPrinter(os.Stdout, printerOptions...).Print(diffResult)
	}
	return nil
}
