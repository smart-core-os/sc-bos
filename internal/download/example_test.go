package download_test

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/smart-core-os/sc-bos/internal/download"
)

// Example demonstrates the end-to-end flow: generating a signed URL,
// fetching the URL and handling the request.
func Example() {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	signer := download.NewHMACSigner(key)

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	router := download.NewRouter(signer,
		download.WithBaseURL(srv.URL+"/download"),
	)
	router.HandleFunc("greeting", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "Hello, %s!\n", download.PayloadFromContext(r.Context()))
	})
	mux.Handle("GET /download/", router)

	// In a real deployment the URL is issued from a gRPC handler that has
	// already authorised the request; here we just call GenerateURL inline.
	dlURL, _, err := router.GenerateURL("greeting", []byte("world"))
	if err != nil {
		panic(err)
	}

	// This part would be performed by the client.
	client := srv.Client()
	resp, err := client.Get(dlURL)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(body))

	// Output: Hello, world!
}
