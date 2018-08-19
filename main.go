package main

import (
	"fmt"
	"time"

	"github.com/coreos/go-iptables/iptables"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	ipt, err := iptables.New()
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Configuring masquerade")
	err = ipt.Append("nat", "POSTROUTING", "-o", "eth0", "-j", "MASQUERADE")
	if err != nil {
		panic(err.Error())
	}

	for {
		// Configure minecraft
		service, _ := clientset.CoreV1().Services("philde").Get("minecraft", metav1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("Minecraft IP is %s\n", service.Spec.ClusterIP)
		err = ipt.AppendUnique("nat", "PREROUTING", "-p", "tcp", "-i", "eth0", "--dport", "25565", "-j", "DNAT", "--to-destination", service.Spec.ClusterIP)
		if err != nil {
			panic(err.Error())
		}

		rules, _ := ipt.List("nat", "PREROUTING")
		for _, rule := range rules {
			fmt.Println(rule)
		}

		time.Sleep(10 * time.Second)
	}
}
