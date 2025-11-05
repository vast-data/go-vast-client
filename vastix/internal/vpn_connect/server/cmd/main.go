// VPN Server - Standalone executable
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"vastix/internal/vpn_connect/common"
	"vastix/internal/vpn_connect/server"
)

func main() {
	// Command-line flags
	port := flag.Uint("port", 51820, "VPN listen port")
	serverIP := flag.String("server-ip", "", "Server VPN IP (e.g., 10.99.1.1)")
	vpnNetwork := flag.String("vpn-network", "", "VPN network CIDR (e.g., 10.99.1.0/24)")
	privateIPs := flag.String("private-ips", "", "Comma-separated list of private IPs to route (e.g., 172.21.101.10,172.21.101.11)")
	iface := flag.String("interface", "", "External network interface (auto-detect if empty)")
	privateKey := flag.String("private-key", "", "Server private key (generate if empty)")
	heartbeatFile := flag.String("heartbeat-file", "", "Heartbeat file path for self-destruction (empty = disabled)")
	logFile := flag.String("log-file", "", "Log file path (empty = stdout)")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	// Set up logger
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	// Determine log output
	logOutput := os.Stdout
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", *logFile, err)
			os.Exit(1)
		}
		defer f.Close()
		logOutput = f
	}

	logger := slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("=== VPN Server Starting ===", slog.Uint64("port", uint64(*port)))

	// Generate keys if not provided
	var privKey, pubKey string
	var err error

	if *privateKey == "" {
		privKey, pubKey, err = common.GenerateKeyPair()
		if err != nil {
			logger.Error("Failed to generate key pair", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Generated new key pair", slog.String("publicKey", pubKey))
	} else {
		privKey = *privateKey
		pubKey, err = common.GetPublicKey(privKey)
		if err != nil {
			logger.Error("Failed to derive public key", slog.Any("error", err))
			os.Exit(1)
		}
	}

	// Auto-detect external interface if not provided
	if *iface == "" {
		detected, err := server.DetectExternalInterface()
		if err != nil {
			logger.Warn("Failed to auto-detect interface", slog.Any("error", err))
			logger.Info("NAT/forwarding will not be configured. Specify -interface manually if needed.")
		} else {
			*iface = detected
			logger.Info("Detected external interface", slog.String("interface", *iface))
		}
	}

	// Parse IPs
	var srvIP netip.Addr
	if *serverIP != "" {
		srvIP, err = netip.ParseAddr(*serverIP)
		if err != nil {
			logger.Error("Invalid server IP", slog.Any("error", err))
			os.Exit(1)
		}
	} else {
		logger.Error("Server IP is required. Use -server-ip flag")
		os.Exit(1)
	}

	var vpnNet netip.Prefix
	if *vpnNetwork != "" {
		vpnNet, err = netip.ParsePrefix(*vpnNetwork)
		if err != nil {
			logger.Error("Invalid VPN network", slog.Any("error", err))
			os.Exit(1)
		}
	} else {
		logger.Error("VPN network is required. Use -vpn-network flag")
		os.Exit(1)
	}

	// Parse private IPs (comma-separated list)
	var privateIPsList []netip.Addr
	if *privateIPs != "" {
		ipStrings := strings.Split(*privateIPs, ",")
		for _, ipStr := range ipStrings {
			ipStr = strings.TrimSpace(ipStr)
			if ipStr == "" {
				continue
			}
			ip, err := netip.ParseAddr(ipStr)
			if err != nil {
				logger.Error("Invalid private IP", slog.String("ip", ipStr), slog.Any("error", err))
				os.Exit(1)
			}
			privateIPsList = append(privateIPsList, ip)
		}
		logger.Info("Parsed private IPs", slog.Int("count", len(privateIPsList)))
	}

	// Create server configuration
	config := &common.ServerConfig{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		ListenPort: uint16(*port),
		ServerIP:   srvIP,
		VPNNetwork: vpnNet,
		PrivateIPs: privateIPsList,
		Interface:  *iface,
	}

	// Create server
	srv, err := server.NewServer(config, logger)
	if err != nil {
		logger.Error("Failed to create server", slog.Any("error", err))
		os.Exit(1)
	}

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		sig := <-sigCh
		logger.Info("Received shutdown signal", slog.String("signal", sig.String()))
		cancel()
	}()

	// Set heartbeat file if provided
	if *heartbeatFile != "" {
		srv.SetHeartbeatFile(*heartbeatFile)
		logger.Info("Heartbeat self-destruction enabled", slog.String("file", *heartbeatFile), slog.String("timeout", "15s"))
	}

	// Log server configuration
	logger.Info("Server configuration",
		slog.String("public_key", pubKey),
		slog.String("server_ip", srvIP.String()),
		slog.Uint64("listen_port", uint64(*port)),
		slog.String("vpn_network", vpnNet.String()),
		slog.Int("private_ips_count", len(privateIPsList)),
	)

	if *iface != "" {
		logger.Info("External interface", slog.String("interface", *iface))
	}

	// Start server
	logger.Info("Starting VPN server...")
	if err := srv.Start(ctx); err != nil {
		logger.Error("Server failed", slog.Any("error", err))
		os.Exit(1)
	}

	// Ensure cleanup happens on ANY exit (crash, kill, etc.)
	defer func() {
		logger.Info("Running cleanup...")
		if err := srv.Stop(); err != nil {
			logger.Error("Failed to stop server", slog.Any("error", err))
		} else {
			logger.Info("Server cleanup completed")
		}
	}()

	logger.Info("Server running, waiting for shutdown signal...")

	// Wait for shutdown signal
	<-ctx.Done()

	logger.Info("=== VPN Server Stopped ===", slog.String("reason", "context cancelled"))
}
