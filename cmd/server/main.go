package main

import (
	"context"
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	ListServices(context.Background(), cli)
}

// ListServices lists services
func ListServices(ctx context.Context, kubeClient kubernetes.Interface) {
	inf := informers.NewSharedInformerFactoryWithOptions(kubeClient, 0)
	serviceInformer := inf.Core().V1().Services()

	inf.Start(ctx.Done())

	services, err := serviceInformer.Lister().Services("default").List(labels.Everything())
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, s := range services {
		fmt.Printf("%+v\n", s)
	}
}
