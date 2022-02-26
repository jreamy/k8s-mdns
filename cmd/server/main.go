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

	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/hashicorp/mdns"
	"github.com/miekg/dns"
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

	nodes, err := cli.CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}

	var services []Service
	for _, n := range nodes.Items {
		var s Service

		fmt.Println(n.Status.Addresses)

		for _, addr := range n.Status.Addresses {
			if ip := net.ParseIP(addr.Address); ip != nil {
				s.IP = ip
			} else {
				s.Hostname = addr.Address
			}
		}

		if s.IP == nil {
			continue
		}

		if s.Hostname == "" {
			s.Hostname = s.IP.String()
		}

		services = append(services, s)
	}

	s, err := ListServices(context.Background(), cli)
	if err != nil {
		log.Fatal(err.Error())
	}

	all := Services(append(services, s...))

	data, _ := json.MarshalIndent(all, "", "  ")
	fmt.Println(string(data))

	// Create the mDNS server, defer shutdown
	server, err := mdns.NewServer(&mdns.Config{
		LogEmptyResponses: true,
		Zone:              &all,
	})
	defer server.Shutdown()
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println("mdns server started")

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

	return out, nil
}

type Services []Service

func (s *Services) Records(q dns.Question) []dns.RR {
	fmt.Printf("%+v\n", q)
	data, _ := json.Marshal(q)
	fmt.Println(string(data))

	const defaultTTL = 120

	if ip := net.ParseIP(dnsutil.ExtractAddressFromReverse(q.Name)); ip != nil {
		for _, s := range Services {
			if s.IP.Equals(ip) {
				fmt.Println("responding to arp request for " + ip.String() + " with " + s.Hostname)
				return []dns.RR{&dns.A{
					Hdr: dns.RR_Header{
						Name:   s.Hostname,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    defaultTTL,
					},
					A: ip,
				}}
			}
		}
	}

	return nil
}
