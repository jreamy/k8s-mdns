package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	svc, err := kubeClient.CoreV1().Services("").List(ctx, v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, s := range svc.Items {
		fmt.Println()

		fmt.Println("name:     ", s.ObjectMeta.Name)
		fmt.Println("namespace:", s.ObjectMeta.Namespace)
		fmt.Println("addr:     ", s.Status.LoadBalancer)

		fmt.Println()

		data, _ := json.MarshalIndent(s, "", "  ")
		fmt.Println(string(data))
	}
}
