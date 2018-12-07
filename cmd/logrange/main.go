// Copyright 2018 The logrange Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/jrivets/log4g"
	"github.com/logrange/logrange/server"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v2"
)

const (
	Version = "0.1.0"
)

var log = log4g.GetLogger("logrange")
var cfg = server.GetDefaultConfig()

func main() {
	defer log4g.Shutdown()

	app := &cli.App{
		Name:    "logrange",
		Version: Version,
		Usage:   "Log Aggregation service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-config-file",
				Usage: "The log4g configuration file name",
				Value: "/opt/logrange/log4g.properties",
			},
			&cli.StringFlag{
				Name:  "config-file",
				Usage: "The logrange configuration file name",
				Value: "/opt/logrange/config.json",
			},
		},
		Before: before,
		Commands: []*cli.Command{
			&cli.Command{
				Name:   "start",
				Usage:  "Run the service",
				Action: runServer,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "host-id",
						Usage: "Unique host identifier, if 0 the id will be automatically assigned.",
						Value: int(cfg.HostHostId),
					},
					&cli.StringFlag{
						Name:  "host-rpc-address",
						Usage: "Advertised RPC address. Peers in the cluster will use it for connecting to the host",
						Value: string(cfg.HostRpcAddress),
					},
					&cli.IntFlag{
						Name:  "host-lease-ttl",
						Usage: "Lease TTL in seconds. Used in cluster config",
						Value: int(cfg.HostLeaseTTLSec),
					},
					&cli.IntFlag{
						Name:  "host-registration-timeout",
						Usage: "Host registration timeout in seconds. 0 means forewer.",
						Value: int(cfg.HostRegisterTimeoutSec),
					},
					&cli.StringFlag{
						Name:  "journals-dir",
						Usage: "Defines path to the journals database directory",
						Value: cfg.JournalsDir,
					},
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Commands[0].Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Run(os.Args)
}

func before(c *cli.Context) error {
	logCfgFile := c.String("log-config-file")
	if logCfgFile != "" {
		if _, err := os.Stat(logCfgFile); os.IsNotExist(err) {
			log.Warn("No file ", logCfgFile, " will use default log4g configuration")
		} else {
			log.Info("Loading log4g config from ", logCfgFile)
			err := log4g.ConfigF(logCfgFile)
			if err != nil {
				err := errors.Wrapf(err, "Could not parse %s file as a log4g configuration, please check syntax ", logCfgFile)
				log.Fatal(err)
				return err
			}
		}
	}

	fc := server.ReadConfigFromFile(c.String("config-file"))
	if fc != nil {
		// overwrite default settings from file
		cfg.Apply(fc)
	}

	return nil
}

func runServer(c *cli.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		select {
		case s := <-sigChan:
			log.Info("Got signal \"", s, "\", cancelling context ")
			cancel()
		}
	}()

	// fill up config

	return server.Start(ctx, cfg)
}