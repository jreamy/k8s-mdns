package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/external-dns/pkg/apis/externaldns"
	"sigs.k8s.io/external-dns/pkg/apis/externaldns/validation"
	"sigs.k8s.io/external-dns/source"
)

func main() {

	cfg1, err := rest.InClusterConfig()
	fmt.Println(cfg1, err)

	cfg := externaldns.NewConfig()
	if err := cfg.ParseFlags(os.Args[1:]); err != nil {
		log.Fatalf("flag parsing error: %v", err)
	}
	log.Printf("config: %s", cfg)

	if err := validation.ValidateConfig(cfg); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	g := &source.SingletonClientGenerator{
		KubeConfig:   cfg.KubeConfig,
		APIServerURL: cfg.APIServerURL,
		// If update events are enabled, disable timeout.
		RequestTimeout: func() time.Duration {
			if cfg.UpdateEvents {
				return 0
			}
			return cfg.RequestTimeout
		}(),
	}

	cli, err := g.KubeClient()
	if err != nil {
		log.Fatalf("client initialization failed: %v", err)
	}

	ListServices(context.Background(), cli)
}

// ListServices lists services
func ListServices(ctx context.Context, kubeClient kubernetes.Interface) {
	inf := informers.NewSharedInformerFactoryWithOptions(kubeClient, 0)
	serviceInformer := inf.Core().V1().Services()

	services, err := serviceInformer.Lister().Services("default").List(labels.Everything())
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, s := range services {
		fmt.Printf("%+v\n", s)
	}
}
