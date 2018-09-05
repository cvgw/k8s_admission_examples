package main

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	appsV1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	vaultApi "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"fmt"
)

const (
	initializerName   = "vault.initializer.cvgw.me"
	requireAnnotation = true
	annotation        = "initializer.cvgw.me/vault"
	vaultAddressVar = "VAULT_ADDRESS"
	vaultTokenVar = "VAULT_TOKEN"
	vaultClientCertPathVar = "VAULT_CLIENT_CERT_PATH"
	keyName = "/secret/foo"
)

var (
	vaultAddress string
	vaultToken string
	vaultClientCertPath string
)

func getKeyFromVault() (string, string, error) {
	config := vaultApi.Config{
		Address: vaultAddress,
	}

	tlsConfig := vaultApi.TLSConfig{
		CACert: vaultClientCertPath,
	}


	config.ConfigureTLS(&tlsConfig)
	client, err := vaultApi.NewClient(&config)
	if err != nil {
		return "", "", err
	}

	client.SetToken(vaultToken)

	secretValues, err := client.Logical().Read(keyName)
	if err != nil {
		return "", "", err
	}

	var name, value string
	for key, val := range secretValues.Data {
		var ok bool
		name = key
		value, ok = val.(string)
		if !ok {
			return "", "", errors.New("couldn't assert as string")
		}
	}

	return name, value, nil
}

func processDeployment(depl *appsV1.Deployment) error {
	name, value, err := getKeyFromVault()
	if err != nil {
		return err
	}

	depl.ObjectMeta.Annotations["vault-initializer"] = fmt.Sprintf("%s-%s", name, value)

	return nil
}

func processAdd(deployment *appsV1.Deployment, clientSet *kubernetes.Clientset) error {
	if deployment.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := deployment.ObjectMeta.GetInitializers().Pending

		if initializerName == pendingInitializers[0].Name {
			log.Info("starting initialization")

			initializedDeployment := deployment.DeepCopy()

			// Remove self from the list of pending Initializers while preserving ordering.
			if len(pendingInitializers) == 1 {
				initializedDeployment.ObjectMeta.Initializers = nil
			} else {
				initializedDeployment.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
			}

			if requireAnnotation {
				a := deployment.ObjectMeta.GetAnnotations()
				log.Infof("Annotations: %s", a)
				_, ok := a[annotation]
				if !ok {
					log.Infof("Required '%s' annotation missing; skipping envoy container injection", annotation)
					_, err := clientSet.AppsV1().Deployments(deployment.Namespace).Update(initializedDeployment)
					if err != nil {
						return err
					}
					return nil
				}
			}

			err := processDeployment(initializedDeployment)
			if err != nil {
				return err
			}

			oldData, err := json.Marshal(deployment)
			if err != nil {
				return err
			}

			newData, err := json.Marshal(initializedDeployment)
			if err != nil {
				return err
			}

			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, appsV1.Deployment{})
			if err != nil {
				return err
			}

			_, err = clientSet.AppsV1().Deployments(deployment.Namespace).Patch(deployment.Name, types.StrategicMergePatchType, patchBytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	ns := "default"
	resource := "deployments"

	vaultAddress = os.Getenv(vaultAddressVar)
	vaultToken = os.Getenv(vaultTokenVar)
	vaultClientCertPath = os.Getenv(vaultClientCertPathVar)

	log.Info("starting initializer")

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	listOpts := metav1.ListOptions{}
	deploymentList, err := clientSet.AppsV1().Deployments(ns).List(listOpts)
	if err != nil {
		log.Fatal(err)
	}

	for _, deployment := range deploymentList.Items {
		log.Info(deployment)
	}

	// Watch uninitialized Deployments in all namespaces.
	restClient := clientSet.AppsV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, resource, ns, fields.Everything())

	// Wrap the returned watchlist to workaround the inability to include
	// the `IncludeUninitialized` list option when setting up watch clients.
	// TODO this can possibly be updated using the list options modifier argument
	includeUninitializedWatchlist := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.IncludeUninitialized = true
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.IncludeUninitialized = true
			return watchlist.Watch(options)
		},
	}

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(includeUninitializedWatchlist, &appsV1.Deployment{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Info("add triggered")
				err := processAdd(obj.(*appsV1.Deployment), clientSet)
				if err != nil {
					log.Error(err)
				}
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Info("Shutdown signal received, exiting...")
	close(stop)

}
