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
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/tdrn-org/go-diff"
	"github.com/tdrn-org/go-log"
	"github.com/tdrn-org/netscanner/internal/buildinfo"
	"github.com/tdrn-org/netscanner/internal/mtls"
)

func RunArgs(ctx context.Context, args []string) error {
	cmdParser, err := kong.New(&cmdLine{}, kong.BindTo(ctx, (*context.Context)(nil)), cmdLineApplication, cmdLineHelpOptions, cmdLineVars)
	if err != nil {
		return err
	}
	cmd, err := cmdParser.Parse(args)
	if err != nil {
		return err
	}
	return cmd.Run()
}

var cmdLineApplication = kong.Name(buildinfo.Cmd())

var cmdLineHelpOptions = kong.ConfigureHelp(kong.HelpOptions{
	Compact: true,
})

var cmdLineVars = kong.Vars{
	"config_default":   DefaultConfigPath(),
	"on_default":       "local",
	"cn_default":       "localhost",
	"validity_default": "8760h", // 1 year
}

type cmdLine struct {
	Silent        bool          `short:"s" help:"Enable silent mode (log level error)"`
	Quiet         bool          `short:"q" help:"Enable quiet mode (log level warn)"`
	Verbose       bool          `short:"v" help:"Enable verbose output (log level info)"`
	Debug         bool          `short:"d" help:"Enable debug output (log level debug)"`
	RunCmd        runCmd        `cmd:"" name:"run" default:"withargs" help:"Run server"`
	VersionCmd    versionCmd    `cmd:"" name:"version" help:"Show version info"`
	TemplateCmd   templateCmd   `cmd:"" name:"template" help:"Output config template"`
	GenTLSCACmd   genTLSCACmd   `cmd:"" name:"generate-tls-ca" help:"Generate CA certificate for mtls based sync"`
	GenTLSNodeCmd genTLSNodeCmd `cmd:"" name:"generate-tls-node" help:"Generate Node certificate for mtls based sync"`
}

type runCmd struct {
	Config string `short:"c" help:"The configuration file to use" default:"${config_default}"`
}

func (cmd *runCmd) Run(ctx context.Context, args *cmdLine) error {
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
	server, err := StartServer(ctx, config)
	if err != nil {
		return err
	}
	stoppedWG := sync.WaitGroup{}
	stoppedWG.Go(func() {
		err = errors.Join(server.Run(ctx), server.Close())
	})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		sigintCtx, cancelListenAndServe := context.WithCancel(ctx)
		go func() {
			<-sigint
			slog.Info("signal SIGINT; stopping")
			cancelListenAndServe()
		}()
		<-sigintCtx.Done()
		server.Shutdown(ctx)
	}()
	stoppedWG.Wait()
	if err == nil {
		slog.Info("stopped")
	}
	return err
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

func (cmd *versionCmd) Run(_ context.Context, args *cmdLine) error {
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

func (cmd *templateCmd) Run(_ context.Context, args *cmdLine) error {
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

type genTLSCACmd struct {
	ON       string        `help:"The organization name to use" default:"${on_default}"`
	Validity time.Duration `help:"The validity time range of the CA certificate" default:"${validity_default}"`
	CRTFile  string        `name:"crt-file" help:"The path/name of the CA certificate file to write"`
	KeyFile  string        `name:"key-file" help:"The path/name of the CA key file to write"`
	Force    bool          `short:"f" help:"Force overwrite of existing files"`
}

func (cmd *genTLSCACmd) Run(_ context.Context, args *cmdLine) error {
	logger := slog.Default()
	log.Notice(logger, "Generating CA certificate")
	if cmd.Force {
		logger.Warn("Overriding any existing file")
	}
	log.Notice(logger, fmt.Sprintf("CRT file    : '%s'", cmd.CRTFile))
	log.Notice(logger, fmt.Sprintf("Organization: '%s'", cmd.ON))
	log.Notice(logger, fmt.Sprintf("Validity    : %s", cmd.Validity.String()))
	options := &mtls.CAOptions{
		CommonOptions: mtls.CommonOptions{
			ON:       cmd.ON,
			Validity: cmd.Validity,
		},
	}
	credentials, err := options.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate CA certificate/key (cause: %w)", err)
	}
	log.Notice(logger, fmt.Sprintf("Expires     : %s", credentials.Certificate.NotAfter.Local().Format(time.RFC3339)))
	err = credentials.Write(cmd.CRTFile, cmd.KeyFile, cmd.Force)
	if err != nil {
		return fmt.Errorf("failed to write CA files (cause: %w)", err)
	}
	log.Notice(logger, "CA certificate successfully generated")
	return nil
}

type genTLSNodeCmd struct {
	ON        string        `help:"The organization name to use" default:"${on_default}"`
	CN        string        `help:"The common name to use" default:"${cn_default}"`
	Validity  time.Duration `help:"The validity time range of the Node certificate" default:"${validity_default}"`
	CRTFile   string        `name:"crt-file" help:"The path/name of the Node certificate file to write"`
	KeyFile   string        `name:"key-file" help:"The path/name of the Node key file to write"`
	CACRTFile string        `name:"ca-crt-file" help:"The path/name to the certificate file of the signing CA"`
	CAKeyFile string        `name:"ca-key-file" help:"The path/name to the key file of the signing CA"`
	Force     bool          `short:"f" help:"Force overwrite of existing files"`
}

func (cmd *genTLSNodeCmd) Run(_ context.Context, args *cmdLine) error {
	logger := slog.Default()
	log.Notice(logger, "Generating Node certificate")
	if cmd.Force {
		logger.Warn("Overriding any existing file")
	}
	log.Notice(logger, fmt.Sprintf("CRT file    : '%s'", cmd.CRTFile))
	log.Notice(logger, fmt.Sprintf("Organization: '%s'", cmd.ON))
	log.Notice(logger, fmt.Sprintf("Common name : '%s'", cmd.CN))
	log.Notice(logger, fmt.Sprintf("Validity    : %s", cmd.Validity.String()))
	caCredentials, err := mtls.LoadCredentials(cmd.CACRTFile, cmd.CAKeyFile, "")
	if err != nil {
		return fmt.Errorf("failed to load CA credentials (cause: %w)", err)
	}
	options := &mtls.NodeOptions{
		CommonOptions: *mtls.InitNodeOptions(cmd.ON, cmd.CN, cmd.Validity, (net.IP).IsPrivate),
		CA:            caCredentials,
	}
	credentials, err := options.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate Node certificate/key (cause: %w)", err)
	}
	log.Notice(logger, fmt.Sprintf("Expires     : %s", credentials.Certificate.NotAfter.Local().Format(time.RFC3339)))
	log.Notice(logger, fmt.Sprintf("IPs         : %v", credentials.Certificate.IPAddresses))
	log.Notice(logger, fmt.Sprintf("DNS         : %v", credentials.Certificate.DNSNames))
	err = credentials.Write(cmd.CRTFile, cmd.KeyFile, cmd.Force)
	if err != nil {
		return fmt.Errorf("failed to write Node files (cause: %w)", err)
	}
	log.Notice(logger, "Node certificate successfully generated")
	return nil
}
