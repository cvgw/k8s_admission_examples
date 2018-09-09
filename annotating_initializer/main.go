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
)

const (
	initializerName = "annotator.initializer.cvgw.me"
)

func handleAdd(deployment *appsV1.Deployment, clientSet *kubernetes.Clientset) error {
	if deployment.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := deployment.ObjectMeta.GetInitializers().Pending

		if initializerName == pendingInitializers[0].Name {
			initializedDeployment := deployment.DeepCopy()

			// Remove initializer name from pending list and preserve order.
			if len(pendingInitializers) == 1 {
				initializedDeployment.ObjectMeta.Initializers = nil
			} else {
				initializedDeployment.ObjectMeta.Initializers.Pending = append(
					pendingInitializers[:0], pendingInitializers[1:]...,
				)
			}

			if initializedDeployment.ObjectMeta.Annotations == nil {
				initializedDeployment.ObjectMeta.Annotations = make(map[string]string)
			}
			// Modify the deployment spec to include the new annotation
			initializedDeployment.ObjectMeta.Annotations["annotating-initializer"] = "meow"

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

			_, err = clientSet.AppsV1().Deployments(deployment.Namespace).Patch(
				deployment.Name, types.StrategicMergePatchType, patchBytes,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	log.Info("starting initializer")

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	restClient := clientSet.AppsV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(
		restClient, "deployments", "default", fields.Everything(),
	)

	// NewListWatchFromClient does not allow IncludeUninitialized to be set
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

				err := handleAdd(obj.(*appsV1.Deployment), clientSet)
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
