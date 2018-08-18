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

	fmt.Println("Hello chains!")
	chains, err := ipt.ListChains("nat")
	if err != nil {
		panic(err.Error())
	}
	for _, chain := range chains {
		fmt.Println(chain)
	}

	for {
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

		time.Sleep(10 * time.Second)
	}
}
