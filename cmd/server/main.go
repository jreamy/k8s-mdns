package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

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

	s, err := ListServices(context.Background(), cli)
	if err != nil {
		log.Fatal(err.Error())
	}
}

type Service struct {
	Hostname string
	IP       net.IP
}

// ListServices lists services
func ListServices(ctx context.Context, kubeClient kubernetes.Interface) ([]Service, error) {
	svc, err := kubeClient.CoreV1().Services("").List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var out []Service
	for _, s := range svc.Items {
		for _, i := range s.Status.LoadBalancer.Ingress {
			if ip := net.ParseIP(i.IP); ip != nil {
				out = append(out, Service{
					Hostname: s.ObjectMeta.Name + ".local",
					IP:       ip,
				})
			}
		}
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))

	return out, nil
}
