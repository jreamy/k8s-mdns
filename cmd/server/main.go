package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/mdns"
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

	fmt.Println(s)

	g := s[0]
	info := []string{"game"}
	service, err := mdns.NewMDNSService(g.IP.String(), "_http._tcp", g.Hostname+".", g.Hostname+".", 80, []net.IP{g.IP}, info)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Create the mDNS server, defer shutdown
	server, err := mdns.NewServer(&mdns.Config{
		Zone:              service,
		LogEmptyResponses: true,
	})
	defer server.Shutdown()
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println("server started")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
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
