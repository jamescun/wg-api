package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	wireguardapi "github.com/jamescun/wg-api"
	"github.com/jamescun/wg-api/server"
	"github.com/jamescun/wg-api/server/jsonrpc"

	flag "github.com/spf13/pflag"
	"golang.zx2c4.com/wireguard/wgctrl"
)

const help = `WG-API presents a JSON-RPC API to a WireGuard device
Usage: wg-api [options]

Helpers:
  --list-devices  list wireguard devices on this system and their name to be
                  given to --device
  --version       display the version number of WG-API

Options:
  --device=<name>         (required) name of WireGuard device to manager
  --listen=<[host:]port>  address where API server will bind
                          (default localhost:8080)
  --tls                   enable Transport Layer Security (SSL) on server
  --tls-key               TLS private key
  --tks-cert              TLS certificate file
  --tls-client-ca         enable mutual TLS authentication (mTLS) of the client
  --token                 opaque value provided by the client to authenticate
                          requests. may be specified multiple times.

Warnings:
  WG-API can perform sensitive network operations, as such it should not be
  publicly exposed. It should be bound to the local interface only, or
  failing that, be behind an authenticating proxy or have mTLS enabled.
`

var (
	// helpers
	listDevices = flag.Bool("list-devices", false, "")
	showVersion = flag.Bool("version", false, "")

	// options
	deviceName  = flag.String("device", "", "")
	listenAddr  = flag.String("listen", "localhost:8080", "")
	enableTLS   = flag.Bool("tls", false, "")
	tlsKey      = flag.String("tls-key", "", "")
	tlsCert     = flag.String("tls-cert", "", "")
	tlsClientCA = flag.String("tls-client-ca", "", "")
	authTokens  = flag.StringArray("token", nil, "")
)

func main() {
	flag.Usage = func() { fmt.Println(help) }
	flag.Parse()

	switch {
	case *listDevices:
		client, err := wgctrl.New()
		if err != nil {
			exitError("could not create WireGuard client: %s", err)
		}

		devices, err := client.Devices()
		if err != nil {
			exitError("could not list WireGuard devices: %s", err)
		}

		if len(devices) > 0 {
			for _, device := range devices {
				fmt.Println(device.Name)
			}
		} else {
			fmt.Println("No WireGuard devices found.")
		}

	case *showVersion:
		fmt.Println("WG-API Version:", wireguardapi.Version)

	default:
		client, err := wgctrl.New()
		if err != nil {
			exitError("could not create WireGuard client: %s", err)
		}

		device, err := client.Device(*deviceName)
		if os.IsNotExist(err) {
			exitError("device %q does not exist", *deviceName)
		} else if err != nil {
			exitError("could not open WireGuard device %q: %s", *deviceName, err)
		}

		svc, err := server.NewServer(client, device.Name)
		if err != nil {
			exitError("could not create WG-API server: %s", err)
		}

		handler := jsonrpc.HTTP(server.Logger(svc))

		if len(*authTokens) > 0 {
			handler = server.AuthTokens(*authTokens...)(handler)
		}

		handler = server.PreventReferer(handler)

		s := &http.Server{
			Addr:    *listenAddr,
			Handler: handler,
		}

		if *enableTLS {
			if *tlsKey == "" || *tlsCert == "" {
				exitError("tls key and cert required for TLS")
			}

			if *tlsClientCA != "" {
				pool, err := loadCertificatePool(*tlsClientCA)
				if err != nil {
					exitError("could not load client ca: %s", err)
				}

				s.TLSConfig = &tls.Config{
					ClientCAs:  pool,
					ClientAuth: tls.RequireAndVerifyClientCert,
				}
			}

			log.Printf("info: server: listening on https://%s\n", s.Addr)

			if err := s.ListenAndServeTLS(*tlsCert, *tlsKey); err != nil {
				log.Fatalln("fatal: server:", err)
			}
		} else {
			log.Printf("info: server: listening on http://%s\n", s.Addr)

			if err := s.ListenAndServe(); err != nil {
				log.Fatalln("fatal: server:", err)
			}
		}
	}

}

func exitError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func loadCertificatePool(filename string) (*x509.CertPool, error) {
	pemBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()

	ok := pool.AppendCertsFromPEM(pemBytes)
	if !ok {
		return nil, fmt.Errorf("error processing pem certificates")
	}

	return pool, nil
}
