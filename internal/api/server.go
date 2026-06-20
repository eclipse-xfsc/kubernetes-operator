package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/eclipse-xfsc/kubernetes-operator/internal/index"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/logging"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/registry"
	"github.com/eclipse-xfsc/kubernetes-operator/internal/runtimeinfo"
)

type ServerConfig struct {
	Address   string
	Version   runtimeinfo.Info
	Inventory *index.Inventory
	Registry  *registry.Registry
	Logger    logging.Logger
}

func NewServer(cfg ServerConfig) *http.Server {
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(v); err != nil {
			http.Error(w, err.Error(), 500)
		}
	}
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) { write(w, cfg.Version) })
	mux.HandleFunc("/modules", func(w http.ResponseWriter, r *http.Request) {
		type dto struct {
			Name         string   `json:"name"`
			Version      string   `json:"version"`
			Types        []string `json:"types"`
			Capabilities []string `json:"capabilities"`
		}
		out := []dto{}
		for _, m := range cfg.Registry.Modules() {
			caps := []string{}
			for _, c := range m.Capabilities() {
				caps = append(caps, string(c))
			}
			out = append(out, dto{Name: m.Name(), Version: m.Version(), Types: m.Types(), Capabilities: caps})
		}
		write(w, out)
	})
	mux.HandleFunc("/types", func(w http.ResponseWriter, r *http.Request) { write(w, cfg.Registry.Types()) })
	mux.HandleFunc("/providers", func(w http.ResponseWriter, r *http.Request) { write(w, cfg.Inventory.Providers()) })
	mux.HandleFunc("/consumers", func(w http.ResponseWriter, r *http.Request) { write(w, cfg.Inventory.Consumers()) })
	mux.HandleFunc("/injections", func(w http.ResponseWriter, r *http.Request) { write(w, cfg.Inventory.Injections()) })
	mux.HandleFunc("/accounts", func(w http.ResponseWriter, r *http.Request) { write(w, cfg.Inventory.Accounts()) })
	mux.HandleFunc("/accounts/by-consumer/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/accounts/by-consumer/")
		parts := strings.Split(rest, "/")
		if len(parts) != 2 {
			http.Error(w, "expected /accounts/by-consumer/{namespace}/{name}", 400)
			return
		}
		write(w, cfg.Inventory.AccountsByConsumer(parts[0], parts[1]))
	})
	mux.HandleFunc("/manifests/requesting-injection", func(w http.ResponseWriter, r *http.Request) { write(w, cfg.Inventory.ManifestsRequestingInjection()) })
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	return &http.Server{Addr: cfg.Address, Handler: logging.HTTPMiddleware(cfg.Logger, mux)}
}
