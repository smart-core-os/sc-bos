// Command xovis-push-data-sample is an example of how to receive Xovis push data.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/smart-core-os/sc-bos/pkg/driver/xovis"
)

var (
	flagPort     int
	flagHTTPS    bool
	flagCertFile string
	flagKeyFile  string
)

func init() {
	flag.IntVar(&flagPort, "port", 1234, "which port to host the server on")
	flag.BoolVar(&flagHTTPS, "https", false, "set to true to use HTTPS instead of HTTP")
	flag.StringVar(&flagCertFile, "cert-file", "", "path to TLS certificate; required when using HTTPS")
	flag.StringVar(&flagKeyFile, "key-file", "", "path to TLS private key; required when using HTTPS")
}

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", flagPort))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "FATAL: can't bind port: %s\n", err.Error())
		os.Exit(1)
	}

	if flagHTTPS {
		err = http.ServeTLS(listener, http.HandlerFunc(handler), flagCertFile, flagKeyFile)
	} else {
		err = http.Serve(listener, http.HandlerFunc(handler))
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "server stopped: %s\n", err.Error())
	}
}

func handler(res http.ResponseWriter, req *http.Request) {
	fmt.Println(" === RECEIVED A NEW REQUEST ===")
	fmt.Println("Headers:")
	for name, values := range req.Header {
		for _, value := range values {
			fmt.Printf("\t%s: %s\n", name, value)
		}
	}
	fmt.Println()
	fmt.Println("Body:")
	body, err := tee(os.Stdout, req.Body)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to read request body: %s\n", err.Error())
		return
	}
	var decoded xovis.PushData
	err = json.Unmarshal(body, &decoded)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to decode request body: %s\n", err.Error())
	}
	fmt.Println()
	fmt.Printf("Request Body (decoded):\n%+#v\n", decoded)
}

func tee(dst io.Writer, src io.Reader) (data []byte, err error) {
	var buf bytes.Buffer
	combinedWriter := io.MultiWriter(dst, &buf)
	_, err = io.Copy(combinedWriter, src)
	return buf.Bytes(), err
}
