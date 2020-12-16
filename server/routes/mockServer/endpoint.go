package mockServer

import (
	"encoding/json"
	"fmt"
	"github.com/cjburchell/loki/models"
	"github.com/cjburchell/uatu-go"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"
)

// endpoint configuration
type endpoint struct {
	models.Endpoint
	responseBody []byte
	log log.ILog
	isVerbose bool
}

func createDefaultEndpoint(ep models.Endpoint, log log.ILog) *endpoint {
	endpoint := &endpoint{Endpoint:ep, log: log}
	endpoint.updateResponseBody()
	return endpoint
}

func (ep *endpoint) updateResponseBody()  {
	if ep.ResponseBody != nil {
		if ep.ContentType == "application/json" {
			ep.setJSONBody(ep.ResponseBody)
		} else {
			var body string
			err := json.Unmarshal(ep.ResponseBody, &body)
			if err == nil {
				ep.setStringBody(body)
			}
		}
	}

	if len(ep.StringBody) != 0 {
		ep.setStringBody(ep.StringBody)
	}
}

func (ep *endpoint) setJSONBody(body interface{}) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		ep.log.Error(err, "Marshal Json")
	}

	ep.setRawBody(bodyBytes)
}

func (ep *endpoint) setRawBody(body []byte) {
	ep.responseBody = body
	ep.log.Printf("Setting Reply Body of %s", ep.responseBody)
}

func (ep *endpoint) setStringBody(body string)  {
	ep.setRawBody([]byte(body))
}

func (ep *endpoint) handleEndpoint(w http.ResponseWriter, r *http.Request) {
	ep.log.Printf("Handling endpoint %s %s %s", ep.Name, ep.Method, ep.Path)

	var bodyString = ""
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err == nil {
		bodyString = string(bodyBytes)
	}

	requestData, _ := json.Marshal(struct {
		Endpoint    string `json:"endpoint"`
		Path        string `json:"path"`
		ContentType string `json:"content_type"`
		Body        string `json:"body"`
	}{
		Endpoint:    ep.Name,
		Path:        ep.Path,
		Body:        bodyString,
		ContentType: r.Header.Get("Content-type"),
	})

	if ep.isVerbose {
		requestDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			ep.log.Error(err, "Unable to dump Request")
		}

		ep.log.Print(string(requestDump))

		vars := mux.Vars(r)
		if len(vars) != 0 {
			ep.log.Print("Values:")
			for key, value := range vars {
				ep.log.Printf("Key: %s, Value: $s", key, value)
			}
		}
	}

	ep.handleReply(w)
	fmt.Printf("Request:%s\n", string(requestData))
}

func (ep endpoint) handleReply(w http.ResponseWriter) {
	if ep.ReplyDelay != 0 {
		ep.log.Printf("Waiting for %sms", ep.ReplyDelay)
		time.Sleep(time.Duration(ep.ReplyDelay) * time.Millisecond)
	}

	ep.log.Printf("Send Response: %d %s Body: %s", ep.Response, ep.ContentType, ep.responseBody)

	if ep.Header != nil {
		for key, value := range ep.Header {
			w.Header().Set(key, value)
		}
	}

	if len(ep.ContentType) != 0 {
		w.Header().Set("Content-Type", ep.ContentType)
	}

	w.WriteHeader(ep.Response)

	if ep.responseBody != nil {
		_, err := w.Write(ep.responseBody)
		if err != nil {
			ep.log.Error(err, "Unable to write response")
		}
	}
}