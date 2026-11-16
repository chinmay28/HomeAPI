package main

import (
	"context"
	"io/fs"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/chinmay28/homeapi/internal/api"
	"github.com/chinmay28/homeapi/internal/db"
	"github.com/chinmay28/homeapi/internal/hostenv"
	"github.com/chinmay28/homeapi/internal/serve"
	"github.com/chinmay28/homeapi/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	var flags serve.Flags

	rootCmd := &cobra.Command{
		Use:     "homeapi",
		Short:   "HomeAPI server",
		Version: version.String(),
		Run: func(cmd *cobra.Command, args []string) {
			// Only does anything where the host needs it (the Android app
			// passes the phone's zone); see internal/hostenv.
			if msg, err := hostenv.Apply(os.Getenv); err != nil {
				log.Printf("Warning: %v; keeping the default time zone", err)
			} else if msg != "" {
				log.Printf("Host: %s", msg)
			}

			cfg, err := serve.Resolve(flags, os.Getenv)
			if err != nil {
				log.Fatalf("Invalid configuration: %v", err)
			}

			dbPath := os.Getenv("HOMEAPI_DB_PATH")

			store, err := db.New(dbPath)
			if err != nil {
				log.Fatalf("Failed to initialize database: %v", err)
			}

			handler := api.NewHandler(store)

			var frontendFS fs.FS
			frontendFS = getEmbeddedFrontend()

			router := api.NewRouter(handler, frontendFS)

			ln, err := net.Listen("tcp", cfg.Addr())
			if err != nil {
				store.Close()
				log.Fatalf("Server failed: %v", err)
			}
			log.Printf("HomeAPI %s starting on http://%s", version.String(), displayAddr(cfg))

			// SIGTERM (systemd, the Android app) and Ctrl-C stop accepting,
			// let running requests finish, then close the database.
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
			defer stop()
			err = serve.Run(ctx, cfg, ln, router, version.String, log.Printf)
			store.Close()
			if err != nil {
				log.Fatalf("Server failed: %v", err)
			}
			log.Printf("HomeAPI stopped")
		},
	}

	rootCmd.Flags().StringVar(&flags.Port, "port", "", "Port to run the server on (overrides HOMEAPI_PORT; default 9999)")
	rootCmd.Flags().StringVar(&flags.Host, "host", "", "Address to bind, e.g. 127.0.0.1 (overrides HOMEAPI_HOST; default every interface)")
	rootCmd.Flags().StringVar(&flags.DiscoveryPort, "discovery-port", "",
		"UDP port to answer network discovery on, 0 to turn it off (overrides HOMEAPI_DISCOVERY_PORT; default 9998; never on a loopback bind)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// displayAddr is the address to print: localhost when bound everywhere, as the
// server always said.
func displayAddr(cfg serve.Config) string {
	if cfg.Host == "" {
		return "localhost" + cfg.Addr()
	}
	return cfg.Addr()
}
