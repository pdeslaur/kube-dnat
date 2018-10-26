package forwarder

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	clientset "github.com/pdeslaur/kube-pat/pkg/client/clientset/versioned"
	informers "github.com/pdeslaur/kube-pat/pkg/client/informers/externalversions/portaddresstranslation/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
)

// Controller is configuring the port forwarding.
type Controller struct {
	opt     ControllerOptions
	pf      *PortForwarder
	s       *Store
	running *atomic.Value
}

// ControllerOptions is a struct for storing configuration options of Controller
type ControllerOptions struct {
	LoadBalancersName map[corev1.Protocol]string
	PatClientSet      *clientset.Clientset
	KubeClientSet     *kubernetes.Clientset
}

// NewController creates a new Controller.
func NewController(
	opt ControllerOptions,
	patInformer informers.PortAddressTranslationInformer,
	serviceInformer corev1informers.ServiceInformer,
) *Controller {
	c := new(Controller)
	c.pf = NewPortForwarderOrDie()
	c.opt = opt
	c.running = new(atomic.Value)
	c.running.Store(false)

	c.s = NewStore(patInformer, serviceInformer, c.Refresh)

	return c
}

// Run starts the controller
func (c Controller) Run(stopCh <-chan struct{}) {
	c.running.Store(true)
	c.Refresh(c.s)

	<-stopCh

	c.running.Store(false)
}

// Refresh updates the Controller configuration
func (c Controller) Refresh(s *Store) error {
	if !c.running.Load().(bool) {
		return nil
	}

	// Clears the current configuration
	err := c.pf.Clear()
	if err != nil {
		return err
	}

	for pfc := range s.Iterate() {
		fmt.Printf("Configuring %s\n", pfc.PortAddressTranslationName)
		err = c.pf.Forward(pfc.Protocol, pfc.SrcPort, pfc.DestIP, pfc.DestPort)
		if err != nil {
			fmt.Printf("Failed to setup forwarding: %s\n", err.Error())
		}
	}

	c.UpdateLoadBalancer(corev1.ProtocolTCP, s)
	c.UpdateLoadBalancer(corev1.ProtocolUDP, s)

	c.pf.Print()

	return nil
}

// UpdateLoadBalancer add ports to the load balancer
func (c Controller) UpdateLoadBalancer(protocol corev1.Protocol, s *Store) error {
	lbName := c.opt.LoadBalancersName[protocol]
	if lbName == "" {
		// The given protocol is not configured
		return nil
	}

	lbNamespace, lbName := split(lbName)
	lbService, err := c.opt.KubeClientSet.CoreV1().Services(lbNamespace).Get(lbName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Failed to fetch the LoadBalancer service for protocol %s", protocol)
	}

	lbPorts := map[int32]bool{}
	for _, port := range lbService.Spec.Ports {
		lbPorts[port.Port] = true
	}

	requiredPorts := map[int32]bool{}
	var servicePorts []corev1.ServicePort

	for pfc := range s.Iterate() {
		if pfc.Protocol != protocol {
			continue
		}
		requiredPorts[pfc.SrcPort] = true

		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     fmt.Sprintf("%s-%d", strings.Replace(pfc.PortAddressTranslationName, "/", "-", -1), pfc.SrcPort),
			Protocol: protocol,
			Port:     pfc.SrcPort,
		})
	}

	if len(requiredPorts) == 0 {
		// Never update the load balancers if no ports are needed. Kubernetes
		// doesn't allow an empty list of ports in services.
		return nil
	}

	if !reflect.DeepEqual(requiredPorts, lbPorts) {
		lbService.Spec.Ports = servicePorts
		fmt.Printf("Updating %s load balancer\n", protocol)
		_, err = c.opt.KubeClientSet.CoreV1().Services(lbNamespace).Update(lbService)
		if err != nil {
			return err
		}
	}

	return nil
}

func split(s string) (namespace, name string) {
	parts := strings.SplitN(s, "/", 2)
	return parts[0], parts[1]
}
