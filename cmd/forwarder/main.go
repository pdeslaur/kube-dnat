package main

import (
	"flag"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	clientset "github.com/pdeslaur/kube-pat/pkg/client/clientset/versioned"
	informers "github.com/pdeslaur/kube-pat/pkg/client/informers/externalversions"
	"github.com/pdeslaur/kube-pat/pkg/forwarder"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	udpService = flag.String("udp-service", "kube-pat/kube-pat-udp", "Name of the service handling incoming UDP traffic")
	tcpService = flag.String("tcp-service", "kube-pat/kube-pat-tcp", "Name of the service handling incoming TCP traffic")
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := SetupSignalHandler()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	pat := clientset.NewForConfigOrDie(cfg)
	kube := kubernetes.NewForConfigOrDie(cfg)

	patInformerFactory := informers.NewSharedInformerFactory(pat, time.Minute)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kube, time.Minute)

	patInformer := patInformerFactory.K8s().V1beta1().PortAddressTranslations()
	coreServiceInformer := kubeInformerFactory.Core().V1().Services()

	opt := forwarder.ControllerOptions{
		LoadBalancersName: map[corev1.Protocol]string{corev1.ProtocolUDP: *udpService, corev1.ProtocolTCP: *tcpService},
		PatClientSet:      pat,
		KubeClientSet:     kube,
	}
	ctrl := forwarder.NewController(opt, patInformer, coreServiceInformer)

	// These are non-blocking.
	fmt.Println("Starting informers...")
	patInformerFactory.Start(stopCh)
	fmt.Println("Waiting for pat")
	patInformerFactory.WaitForCacheSync(stopCh)
	kubeInformerFactory.Start(stopCh)
	fmt.Println("Waiting for kube")
	kubeInformerFactory.WaitForCacheSync(stopCh)

	ctrl.Run(stopCh)
}
