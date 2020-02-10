package main

import (
	"log"
	"net/http"
	"path/filepath"
)

const (
	TlsDir      string = "/run/secrets/tls"
	TlsCertFile string = "tls.crt"
	TlsKeyFile  string = "tls.key"
)

const (
	MutateApi   string = "/mutate"
	WebhookPort string = ":8443"
)

func main() {
	certPath := filepath.Join(TlsDir, TlsCertFile)
	keyPath := filepath.Join(TlsDir, TlsKeyFile)

	mux := http.NewServeMux()
	mux.Handle(MutateApi, admitFuncHandler(applyPodOutput))
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    WebhookPort,
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}
