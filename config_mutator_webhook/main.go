package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/mattbaird/jsonpatch"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	certFilePathVar      = "CERT_FILE_PATH"
	serverKeyFilePathVar = "SERVER_KEY_FILE_PATH"
	portVar              = "PORT"
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

	ar, err := reviewFromBody(body)
	if err != nil {
		log.Error(err)
	}

	response, err := processAdmissionRequest(ar)
	if err != nil {
		log.Error(err)
	}

	resp, err := json.Marshal(response)
	if err != nil {
		log.Error(err)
	}
	_, err = w.Write(resp)
	if err != nil {
		log.Error(err)
	}
}

func reviewFromBody(body []byte) (v1beta1.AdmissionReview, error) {
	ar := v1beta1.AdmissionReview{}
	deserializer := scheme.Codecs.UniversalDeserializer()

	_, _, err := deserializer.Decode(body, nil, &ar)
	if err != nil {
		err = errors.Wrap(err, "couldn't decode webhook body")
		return ar, err
	}

	return ar, nil
}

func processAdmissionRequest(ar v1beta1.AdmissionReview) (v1beta1.AdmissionReview, error) {
	reviewResponse := &v1beta1.AdmissionResponse{
		Allowed: true,
	}
	response := v1beta1.AdmissionReview{}
	response.Response = reviewResponse

	req := ar.Request

	pod, err := podFromRequest(req)
	if err != nil {
		err = errors.Wrap(err, "couldn't get pod from request")
		return response, err
	}

	patchBytes, patchType, err := addConfigToPod(&pod)
	if err != nil {
		err = errors.Wrap(err, "couldn't generate patch")
		return response, err
	}

	reviewResponse.Patch = patchBytes
	reviewResponse.PatchType = &patchType
	response.Response.UID = ar.Request.UID

	return response, nil
}

func podFromRequest(req *v1beta1.AdmissionRequest) (coreV1.Pod, error) {
	var pod coreV1.Pod

	err := json.Unmarshal(req.Object.Raw, &pod)
	if err != nil {
		err = errors.Wrap(err, "couldn't unmarshall pod")
		return pod, err
	}

	return pod, nil
}

func addConfigToPod(pod *coreV1.Pod) ([]byte, v1beta1.PatchType, error) {
	patchType := v1beta1.PatchTypeJSONPatch

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	updatedPod := pod.DeepCopy()
	updatedPod.Annotations["test-mutate-annotation"] = "meow"

	oldData, err := json.Marshal(pod)
	if err != nil {
		return nil, patchType, err
	} else {
		log.Infof("old data %s", string(oldData))
	}

	newData, err := json.Marshal(updatedPod)
	if err != nil {
		return nil, patchType, err
	} else {
		log.Infof("new data %s", string(newData))
	}

	patch, err := jsonpatch.CreatePatch(oldData, newData)
	if err != nil {
		return nil, patchType, err
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return nil, patchType, err
	}

	return patchBytes, patchType, nil
}

func main() {
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

	whsvr := &WebhookServer{
		server: &http.Server{
			Addr:      fmt.Sprintf(":%v", port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.handleWebhook)
	whsvr.server.Handler = mux
	log.Fatal(whsvr.server.ListenAndServeTLS("", ""))
}
