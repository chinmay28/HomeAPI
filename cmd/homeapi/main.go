package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/chinmay28/homeapi/internal/api"
	"github.com/chinmay28/homeapi/internal/db"
	"github.com/spf13/cobra"
)

func main() {
	var port string

	rootCmd := &cobra.Command{
		Use:   "homeapi",
		Short: "HomeAPI server",
		Run: func(cmd *cobra.Command, args []string) {
			// Resolve port: flag > env > default
			if port == "" {
				port = os.Getenv("HOMEAPI_PORT")
			}
			if port == "" {
				port = "8080"
			}

			dbPath := os.Getenv("HOMEAPI_DB_PATH")

			store, err := db.New(dbPath)
			if err != nil {
				log.Fatalf("Failed to initialize database: %v", err)
			}
			defer store.Close()

			handler := api.NewHandler(store)

			var frontendFS fs.FS
			frontendFS = getEmbeddedFrontend()

			router := api.NewRouter(handler, frontendFS)

			addr := fmt.Sprintf(":%s", port)
			log.Printf("HomeAPI starting on http://localhost%s", addr)

			if err := http.ListenAndServe(addr, router); err != nil {
				log.Fatalf("Server failed: %v", err)
			}
		},
	}

	rootCmd.Flags().StringVar(&port, "port", "", "Port to run the server on (overrides HOMEAPI_PORT)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
