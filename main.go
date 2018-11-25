package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/summerwind/cloudevents-gateway/config"
	"github.com/summerwind/cloudevents-gateway/webhook"
	"github.com/summerwind/cloudevents-gateway/webhook/github"
	yaml "gopkg.in/yaml.v2"
)

var (
	VERSION = "0.0.1"
	COMMIT  = "HEAD"
)

// loadConfig loads the specified configuration file and returns
// config.
func loadConfig(configPath string) (*config.Config, error) {
	buf, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	c := config.New()
	err = yaml.Unmarshal(buf, &c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func newProxyHandler(backend *url.URL, parser webhook.Parser) (*httputil.ReverseProxy, error) {
	director := func(req *http.Request) {
		// Copy request body
		body := req.Body
		if body != http.NoBody {
			var buf bytes.Buffer

			_, err := buf.ReadFrom(body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to read request body: %s", err)
				return
			}

			err = body.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s", err)
				return
			}

			body = ioutil.NopCloser(&buf)
			req.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
		}

		ce, err := parser.Parse(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse error: %s", err)
			return
		}

		req.Body = body

		req.Host = backend.Host
		req.URL.Scheme = backend.Scheme
		req.URL.Host = backend.Host
		req.URL.Path = backend.Path

		req.Header.Set("CE-CloudEventsVersion", "0.1")
		req.Header.Set("CE-EventID", ce.EventID)
		req.Header.Set("CE-EventTime", ce.EventTime.Format(time.RFC3339))
		req.Header.Set("CE-EventType", ce.EventType)
		req.Header.Set("CE-EventTypeVersion", ce.EventTypeVersion)
		req.Header.Set("CE-Source", ce.Source)

		if ce.SchemaURL != "" {
			req.Header.Set("CE-SchemaURL", ce.SchemaURL)
		}
		if ce.ContentType != "" {
			req.Header.Set("Content-Type", ce.ContentType)
		}

		log.Printf("remote_addr:%s event_id:%s event_type:%s source:%s", req.RemoteAddr, ce.EventID, ce.EventType, ce.Source)
	}

	return &httputil.ReverseProxy{Director: director}, nil
}

// run starts the HTTP server to process authentication.
func run(cmd *cobra.Command, args []string) error {
	v, err := cmd.Flags().GetBool("version")
	if err != nil {
		return err
	}

	if v {
		fmt.Printf("%s (%s)\n", VERSION, COMMIT)
		return nil
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	c, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	if c.GitHub.Backend != "" {
		backend, err := url.Parse(c.GitHub.Backend)
		if err != nil {
			return err
		}
		parser := github.NewParser(c.GitHub.Secret)

		handler, err := newProxyHandler(backend, parser)
		if err != nil {
			return err
		}

		mux.Handle(c.GitHub.Path, handler)
	}

	server := &http.Server{
		Addr:    c.Listen,
		Handler: mux,
	}

	go func() {
		if c.TLS.CertFile != "" {
			server.ListenAndServeTLS(c.TLS.CertFile, c.TLS.KeyFile)
		} else {
			server.ListenAndServe()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var cmd = &cobra.Command{
		Use:   "cloudevents-gateway",
		Short: "Wehbook gateway for CloudEvents.",
		RunE:  run,

		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.Flags().StringP("config", "c", "config.yml", "Path to the configuration file")
	cmd.Flags().BoolP("version", "v", false, "Display version information and exit")

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}