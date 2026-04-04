package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	yaitracker "yaitracker.com/loweryaustin"
	mcpserver "yaitracker.com/loweryaustin/internal/mcp"
	"yaitracker.com/loweryaustin/internal/server"
	"yaitracker.com/loweryaustin/internal/store"
	mcpstdio "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "yaitracker",
	Short: "YAITracker - Issue tracker with time tracking and MCP server",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("yaitracker %s (commit %s, built %s)\n", version, commit, date)
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	RunE:  runServe,
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server (stdio)",
	RunE:  runMCP,
}

var (
	flagDBPath string
	flagAddr   string
	flagSecret string
	flagCORS   string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&flagDBPath, "db", envOrDefault("YAITRACKER_DB", "yaitracker.db"), "Database file path")
	rootCmd.PersistentFlags().StringVar(&flagSecret, "secret", "", "Application secret (env: YAITRACKER_SECRET)")

	serveCmd.Flags().StringVar(&flagAddr, "addr", envOrDefault("YAITRACKER_ADDR", ":8080"), "Listen address")
	serveCmd.Flags().StringVar(&flagCORS, "cors", envOrDefault("YAITRACKER_CORS_ORIGINS", ""), "Allowed CORS origins")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getSecret() string {
	if flagSecret != "" {
		return flagSecret
	}
	return os.Getenv("YAITRACKER_SECRET")
}

func validateSecret() error {
	secret := getSecret()
	if len(secret) < 32 {
		return fmt.Errorf("YAITRACKER_SECRET must be at least 32 characters (got %d). Set via environment variable or --secret flag", len(secret))
	}
	return nil
}

func runServe(cmd *cobra.Command, args []string) error {
	if err := validateSecret(); err != nil {
		return err
	}

	st, err := store.New(flagDBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer st.Close()

	if err := store.Migrate(st.DB(), yaitracker.MigrationsFS, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cleanupLoop(ctx, st)

	mcpSrv := mcpserver.NewMCPServer(st)
	srv := server.New(st, getSecret(), flagCORS, mcpSrv)
	httpServer := &http.Server{
		Addr:         flagAddr,
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("YAITracker listening on %s", flagAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return httpServer.Shutdown(shutdownCtx)
}

func runMCP(cmd *cobra.Command, args []string) error {
	st, err := store.New(flagDBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer st.Close()

	if err := store.Migrate(st.DB(), yaitracker.MigrationsFS, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	s := mcpserver.NewMCPServer(st)
	return mcpstdio.ServeStdio(s)
}

func cleanupLoop(ctx context.Context, st *store.Store) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := st.CleanExpiredSessions(ctx); err == nil && n > 0 {
				log.Printf("Cleaned %d expired sessions", n)
			}
			if n, err := st.CleanExpiredTokens(ctx); err == nil && n > 0 {
				log.Printf("Cleaned %d expired tokens", n)
			}
			if n, err := st.StopOrphanedTimers(ctx, 8*time.Hour); err == nil && n > 0 {
				log.Printf("Stopped %d orphaned timers", n)
			}
		}
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
