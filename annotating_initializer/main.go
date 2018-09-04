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
	initializerName   = "annotator.initializer.cvgw.me"
	requireAnnotation = true
	annotation        = "initializer.cvgw.me/annotating"
)

func foo(deployment *appsV1.Deployment, clientSet *kubernetes.Clientset) error {
	if deployment.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := deployment.ObjectMeta.GetInitializers().Pending

		if initializerName == pendingInitializers[0].Name {
			//v1beta1.Deployment{}.Dee
			//o, err := runtime.NewScheme().DeepCopy(deployment)
			//if err != nil {
			//	return err
			//}
			//initializedDeployment := o.(*v1beta1.Deployment)
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

			// Modify the Deployment's Pod template to include the Envoy container
			// and configuration volume. Then patch the original deployment.
			//initializedDeployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, c.Containers...)
			//initializedDeployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, c.Volumes...)
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
				err := foo(obj.(*appsV1.Deployment), clientSet)
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