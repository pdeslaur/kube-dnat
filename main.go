package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/coreos/go-iptables/iptables"
	patv1beta1 "github.com/pdeslaur/kube-pat/pkg/apis/portaddresstranslation/v1beta1"
	portaddresstranslation "github.com/pdeslaur/kube-pat/pkg/client/clientset/versioned"
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
	patclientset, err := portaddresstranslation.NewForConfig(config)
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
		pats, err := patclientset.K8sV1beta1().PortAddressTranslations("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		fmt.Println("Clearing the chain")
		err = ipt.ClearChain("nat", "PREROUTING")
		if err != nil {
			panic(err.Error())
		}

		for _, pat := range pats.Items {
			configurePortAddressTranslation(&pat, clientset, ipt)
		}

		rules, _ := ipt.List("nat", "PREROUTING")
		for _, rule := range rules {
			fmt.Println(rule)
		}

		time.Sleep(10 * time.Second)
	}
}

func configurePortAddressTranslation(pat *patv1beta1.PortAddressTranslation, clientset *kubernetes.Clientset, ipt *iptables.IPTables) {
	fmt.Printf("Configuring service %s/%s\n", pat.Namespace, pat.Spec.Service)
	service, err := clientset.CoreV1().Services(pat.Namespace).Get(pat.Spec.Service, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	port := service.Spec.Ports[0]
	address := fmt.Sprintf("%s:%d", service.Spec.ClusterIP, port.Port)
	fmt.Printf("%s IP is %s\n", pat.Spec.Service, address)
	err = ipt.AppendUnique("nat", "PREROUTING", "-p", string(port.Protocol), "-i", "eth0", "--dport", strconv.Itoa(pat.Spec.Port), "-j", "DNAT", "--to-destination", address)
	if err != nil {
		panic(err.Error())
	}
}
