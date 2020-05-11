package agent

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"go.undefinedlabs.com/scopeagent/tags"
)

// Loads the remote agent configuration from local cache, if not exists then retrieve it from the server
func (a *Agent) loadRemoteConfiguration() map[string]interface{} {
	if a == nil || a.metadata == nil {
		return nil
	}
	configRequest := map[string]interface{}{}
	addElementToMapIfEmpty(configRequest, tags.Repository, a.metadata[tags.Repository])
	addElementToMapIfEmpty(configRequest, tags.Commit, a.metadata[tags.Commit])
	addElementToMapIfEmpty(configRequest, tags.Branch, a.metadata[tags.Branch])
	addElementToMapIfEmpty(configRequest, tags.Service, a.metadata[tags.Service])
	addElementToMapIfEmpty(configRequest, tags.Dependencies, a.metadata[tags.Dependencies])
	if cKeys, ok := a.metadata[tags.ConfigurationKeys]; ok {
		cfgKeys := cKeys.([]string)
		configRequest[tags.ConfigurationKeys] = cfgKeys
		for _, item := range cfgKeys {
			addElementToMapIfEmpty(configRequest, item, a.metadata[item])
		}
	}
	if a.debugMode {
		jsBytes, _ := json.Marshal(configRequest)
		a.logger.Printf("Getting remote configuration for: %v", string(jsBytes))
	}
	return a.getOrSetLocalCacheData(configRequest, "remote", false, a.getRemoteConfiguration).(map[string]interface{})
}

// Gets the remote agent configuration from the endpoint + api/agent/config
func (a *Agent) getRemoteConfiguration(cfgRequest map[string]interface{}) interface{} {
	client := &http.Client{}
	curl := a.getUrl("api/agent/config")
	payload, err := msgPackEncodePayload(cfgRequest)
	if err != nil {
		a.logger.Printf("Error encoding payload: %v", err)
	}
	payloadBytes := payload.Bytes()

	var (
		lastError  error
		status     string
		statusCode int
		bodyData   []byte
	)
	for i := 0; i <= numOfRetries; i++ {
		req, err := http.NewRequest("POST", curl, bytes.NewBuffer(payloadBytes))
		if err != nil {
			a.logger.Printf("Error creating new request: %v", err)
			return nil
		}
		req.Header.Set("User-Agent", a.userAgent)
		req.Header.Set("Content-Type", "application/msgpack")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("X-Scope-ApiKey", a.apiKey)

		if a.debugMode {
			if i == 0 {
				a.logger.Println("sending payload")
			} else {
				a.logger.Printf("sending payload [retry %d]", i)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			if v, ok := err.(*url.Error); ok {
				// Don't retry if the error was due to TLS cert verification failure.
				if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
					a.logger.Printf("error: http client returns: %s", err.Error())
					return nil
				}
			}

			lastError = err
			a.logger.Printf("client error '%s', retrying in %d seconds", err.Error(), retryBackoff/time.Second)
			time.Sleep(retryBackoff)
			continue
		}

		statusCode = resp.StatusCode
		status = resp.Status
		if resp.Body != nil && resp.Body != http.NoBody {
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				bodyData = body
			}
		}
		if err := resp.Body.Close(); err != nil { // We can't defer inside a for loop
			a.logger.Printf("error: closing the response body. %s", err.Error())
		}

		if statusCode == 0 || statusCode >= 400 {
			lastError = errors.New(fmt.Sprintf("error from API [status: %s]: %s", status, string(bodyData)))
		}

		// Check the response code. We retry on 500-range responses to allow
		// the server time to recover, as 500's are typically not permanent
		// errors and may relate to outages on the server side. This will catch
		// invalid response codes as well, like 0 and 999.
		if statusCode == 0 || (statusCode >= 500 && statusCode != 501) {
			a.logger.Printf("error: [status code: %d], retrying in %d seconds", statusCode, retryBackoff/time.Second)
			time.Sleep(retryBackoff)
			continue
		}

		if i > 0 {
			a.logger.Printf("payload was sent successfully after retry.")
		}
		break
	}

	if statusCode != 0 && statusCode < 400 && lastError == nil {
		var resp map[string]interface{}
		if err := json.Unmarshal(bodyData, &resp); err == nil {
			return resp
		} else {
			a.logger.Printf("Error unmarshalling json: %v", err)
		}
	}
	return nil
}
