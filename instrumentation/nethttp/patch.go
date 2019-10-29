package nethttp

import (
	"net/http"
	"sync"
)

var once sync.Once

func PatchHttpDefaultClient() {
	once.Do(func() {
		http.DefaultClient = &http.Client{Transport: &Transport{RoundTripper: http.DefaultTransport}}
	})
}

func PatchHttpDefaultClientWithPayload() {
	once.Do(func() {
		http.DefaultClient = &http.Client{Transport: &Transport{
			RoundTripper:           http.DefaultTransport,
			PayloadInstrumentation: true,
		}}
	})
}
