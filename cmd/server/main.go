package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {

	ifiFlag := flag.String("ifi", "en0", "network interface to use")

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
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
				s.Hostname = addr.Address + ".local."
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

	ifi, err := net.InterfaceByName(*ifiFlag)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Create the mDNS server, defer shutdown
	// server, err := mdns.NewServer(&mdns.Config{
	// 	LogEmptyResponses: true,
	// 	Iface:             ifi,
	// 	Zone:              &all,
	// })
	// defer server.Shutdown()
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }

	fmt.Println("mdns server started")

	l, err := Listener(ifi, net.ParseIP("224.0.0.251"), 5353)
	if err != nil {
		log.Fatal(err.Error())
	}

	defer l.LeaveGroup(ifi, &net.UDPAddr{IP: net.ParseIP("224.0.0.251")})

	go func() {
		for {
			data := make([]byte, 1500)
			n, _, src, err := l.ReadFrom(data)
			if err != nil {
				fmt.Println("got err", err)
				continue
			}

			var msg dns.Msg
			if err := msg.Unpack(data[:n]); err != nil {
				log.Printf("[ERR] mdns: Failed to unpack packet: %v", err)
				continue
			}

			fmt.Printf("read %d bytes from %s (%+v)\n", n, src, msg)
			fmt.Println(string(data[:n]))
		}
	}()

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
					Hostname: s.ObjectMeta.Name + ".service.local.",
					IP:       ip,
				})
			}
		}
	}

	return out, nil
}

func Listener(ifi *net.Interface, group net.IP, port int) (*ipv4.PacketConn, error) {

	c, err := net.ListenPacket("udp4", ":"+strconv.Itoa(port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	pkt := ipv4.NewPacketConn(c)
	if err := pkt.JoinGroup(ifi, &net.UDPAddr{IP: group}); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to join group (%s): %w", group, err)
	}

	return pkt, nil
}

type Services []Service

func (s *Services) Records(q dns.Question) []dns.RR {
	fmt.Printf("%+v\n", q)
	data, _ := json.Marshal(q)
	fmt.Println(string(data))

	const defaultTTL = 120

	// var allRecords []dns.RR
	// for _, srv := range *s {
	// 	allRecords = append(allRecords, &dns.A{
	// 		Hdr: dns.RR_Header{
	// 			Name:   srv.Hostname,
	// 			Rrtype: dns.TypeA,
	// 			Class:  dns.ClassINET,
	// 			Ttl:    defaultTTL,
	// 		},
	// 		A: srv.IP,
	// 	})
	// }

	if ip := net.ParseIP(dnsutil.ExtractAddressFromReverse(q.Name)); ip != nil {
		for _, s := range *s {
			if s.IP.Equal(ip) {
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
				// return allRecords
			}
		}
	}

	return nil
}
