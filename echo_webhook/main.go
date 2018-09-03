package main

import (
	"k8s.io/api/admission/v1beta1"
	log "github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"encoding/json"
	"net/http"
	"fmt"
	"io/ioutil"
	"crypto/tls"
	"os"
)

const (
	certFilePathVar = "CERT_FILE_PATH"
	serverKeyFilePathVar = "SERVER_KEY_FILE_PATH"
	portVar = "PORT"
)


type WebhookServer struct {
	server *http.Server
}

func (ws *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	response := processWebhook(body)
	resp, err := json.Marshal(response)
	if err != nil {
		log.Error(err)
	}
	_, err = w.Write(resp)
	if err != nil {
		log.Error(err)
	}
}

func processWebhook(body []byte) v1beta1.AdmissionReview {
	reviewResponse := &v1beta1.AdmissionResponse{
		Allowed: true,
	}

	ar := v1beta1.AdmissionReview{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	_, _, err := deserializer.Decode(body, nil, &ar)
	if err != nil {
		log.Error(err)
	}

	var pod coreV1.Pod
	req := ar.Request

	err = json.Unmarshal(req.Object.Raw, &pod)
	if err != nil {
		log.Error(err)
	}

	log.Info(&pod.ObjectMeta)

	response := v1beta1.AdmissionReview{}
	response.Response = reviewResponse
	response.Response.UID = ar.Request.UID

	return response

}

func main () {
	certFilePath := os.Getenv(certFilePathVar)
	serverKeyFilePath := os.Getenv(serverKeyFilePathVar)
	port := os.Getenv(portVar)

	switch {
	case certFilePath == "":
		log.Fatal("cert file path cannot be blank")
	case serverKeyFilePath == "":
		log.Fatal("server key file path cannot be blank")
	}

	pair, err := tls.LoadX509KeyPair(certFilePath, serverKeyFilePath)
	if err != nil {
		log.Fatal(err)
	}

	whsvr := &WebhookServer {
		server:           &http.Server {
			Addr:        fmt.Sprintf(":%v", port),
			TLSConfig:   &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.handleWebhook)
	whsvr.server.Handler = mux
	log.Fatal(whsvr.server.ListenAndServeTLS("", ""))
}
