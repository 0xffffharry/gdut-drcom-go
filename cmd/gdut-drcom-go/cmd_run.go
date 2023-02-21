package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gdut-drcom-go/config"
	"gdut-drcom-go/core"
	log2 "gdut-drcom-go/lib/log"
	"github.com/spf13/cobra"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var runCommand = &cobra.Command{
	Use:   "run",
	Short: "Run the client",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

var (
	paramConfig string
	//
	paramRemoteIP       string
	paramRemotePort     uint16
	paramKeepAlive1Flag uint8
	paramEnableCrypt    bool
	paramBindDevice     string
	paramLogFile        string
	paramDebug          bool
)

func init() {
	runCommand.Flags().StringVarP(&paramConfig, "config", "c", "", "Config File")
	//
	runCommand.Flags().StringVarP(&paramRemoteIP, "remote-ip", "i", "", "Remote IP")
	runCommand.Flags().Uint16VarP(&paramRemotePort, "port", "p", 61440, "Remote Port")
	runCommand.Flags().Uint8VarP(&paramKeepAlive1Flag, "keep-alive1-flag", "k", 0xdc, "KeepAlive1 Flag")
	runCommand.Flags().BoolVarP(&paramEnableCrypt, "enable-crypt", "e", false, "Enable Crypt")
	runCommand.Flags().StringVarP(&paramBindDevice, "bind-device", "b", "", "Bind Device")
	runCommand.Flags().StringVarP(&paramLogFile, "log-file", "f", "", "Log File")
	runCommand.Flags().BoolVarP(&paramDebug, "debug", "d", false, "Debug Mode")
	RootCommand.AddCommand(runCommand)
}

func run() {
	if paramConfig != "" {
		runMulti()
	} else {
		runSimple()
	}
}

func runSimple() {
	if paramRemoteIP == "" {
		log.Fatalln("remote ip is required")
	}
	remoteIP, err := netip.ParseAddr(paramRemoteIP)
	if err != nil {
		log.Fatalln(fmt.Sprintf("invalid remote ip: %s %s", paramRemoteIP, err))
	}
	if !remoteIP.IsValid() {
		log.Fatalln(fmt.Sprintf("invalid remote ip: %s", paramRemoteIP))
	}
	if paramRemotePort == 0 || paramRemotePort > 65535 {
		log.Fatalln(fmt.Sprintf("invalid remote port: %d", paramRemotePort))
	}
	//
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := log2.NewLogger(os.Stdout, nil)
	if paramLogFile != "" {
		os.Remove(paramLogFile)
		f, err := os.Create(paramLogFile)
		if err != nil {
			log.Fatalln(fmt.Sprintf("open log file failed: %s", err))
		}
		defer f.Close()
		f.Chmod(0666)
		logger.SetOutput(f)
	}
	if paramDebug {
		logger.SetDebug(true)
	}
	logger.Info("global", fmt.Sprintf("gdut-drcom-go %s", Version))
	defer logger.Info("global", "Bye")
	go listenSignal(func() {
		logger.Warn("global", "receive signal, exit")
		cancel()
	})
	d := &core.Drcom{
		Tag:            "core",
		RemoteIP:       remoteIP,
		RemotePort:     paramRemotePort,
		KeepAlive1Flag: paramKeepAlive1Flag,
		EnableCrypt:    paramEnableCrypt,
		BindDevice:     paramBindDevice,
	}
	d.SetLogger(logger)
	err = d.RunWithContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func runMulti() {
	raw, err := os.ReadFile(paramConfig)
	if err != nil {
		log.Fatalln(fmt.Sprintf("read config file failed: %s", err))
	}
	var cfg config.Config
	err = json.Unmarshal(raw, &cfg)
	if err != nil {
		log.Fatalln(fmt.Sprintf("parse config file failed: %s", err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := log2.NewLogger(os.Stdout, nil)
	if cfg.LogFile != "" {
		os.Remove(cfg.LogFile)
		f, err := os.Create(cfg.LogFile)
		if err != nil {
			log.Fatalln(fmt.Sprintf("open log file failed: %s", err))
		}
		defer f.Close()
		f.Chmod(0666)
		logger.SetOutput(f)
	}
	if cfg.Debug {
		logger.SetDebug(true)
	}
	logger.Info("global", fmt.Sprintf("gdut-drcom-go %s", Version))
	defer logger.Info("global", "Bye")
	go listenSignal(func() {
		logger.Warn("global", "receive signal, exit")
		cancel()
	})
	wg := sync.WaitGroup{}
	for _, d := range cfg.Core {
		c := &core.Drcom{
			Tag:            d.Tag,
			RemoteIP:       d.RemoteIP,
			RemotePort:     d.RemotePort,
			KeepAlive1Flag: d.KeepAlive1Flag,
			EnableCrypt:    d.EnableCrypt,
			BindDevice:     d.BindDevice,
		}
		c.SetLogger(logger)
		wg.Add(1)
		go func(c *core.Drcom) {
			defer wg.Done()
			c.RunWithContext(ctx)
		}(c)
	}
	wg.Wait()
}

func listenSignal(do func()) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	do()
}
