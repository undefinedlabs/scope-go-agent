package scopeagent

import (
	"github.com/undefinedlabs/go-agent/instrumentation/nethttp"
	"net/http"
)

func PatchAll() error {
	err := PatchHttpDefaultClient()
	if err != nil {
		return err
	}
	return nil
}

func PatchHttpDefaultClient() error {
	http.DefaultClient = &http.Client{Transport: &nethttp.Transport{}}
	return nil
}
