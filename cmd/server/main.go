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

	"sigs.k8s.io/external-dns/pkg/apis/externaldns"
	"sigs.k8s.io/external-dns/pkg/apis/externaldns/validation"
)

func main() {
	cfg := externaldns.NewConfig()
	if err := cfg.ParseFlags(os.Args[1:]); err != nil {
		log.Fatalf("flag parsing error: %v", err)
	}
	log.Infof("config: %s", cfg)

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

	ListServices(context.Background(), g.KubeClient())
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
		fmt.Println("%+v\n", s)
	}
}
