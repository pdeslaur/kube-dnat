package forwarder

import (
	"fmt"
	"strings"
	"sync/atomic"

	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/api/core/v1"

	patv1beta1 "github.com/pdeslaur/kube-pat/pkg/apis/portaddresstranslation/v1beta1"
	clientset "github.com/pdeslaur/kube-pat/pkg/client/clientset/versioned"
	informers "github.com/pdeslaur/kube-pat/pkg/client/informers/externalversions/portaddresstranslation/v1beta1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Controller is configuring the port forwarding.
type Controller struct {
	opt             ControllerOptions
	pf              *PortForwarder
	patInformer     informers.PortAddressTranslationInformer
	serviceInformer corev1informers.ServiceInformer
	running         *atomic.Value
}

// ControllerOptions is a struct for storing configuration options of Controller
type ControllerOptions struct {
	UDPService    string
	TCPService    string
	PatClientSet  *clientset.Clientset
	KubeClientSet *kubernetes.Clientset
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
	c.patInformer = patInformer
	c.serviceInformer = serviceInformer
	c.running = new(atomic.Value)
	c.running.Store(false)

	patInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.Refresh()
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if oldObj.(*patv1beta1.PortAddressTranslation).GetResourceVersion() != newObj.(*patv1beta1.PortAddressTranslation).GetResourceVersion() {
					c.Refresh()
				}
			},
			DeleteFunc: func(obj interface{}) {
				c.Refresh()
			},
		})

	serviceInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.Refresh()
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if oldObj.(*v1.Service).GetResourceVersion() != newObj.(*v1.Service).GetResourceVersion() {
					c.Refresh()
				}
			},
			DeleteFunc: func(obj interface{}) {
				c.Refresh()
			},
		})

	return c
}

// Run starts the controller
func (c Controller) Run(stopCh <-chan struct{}) {
	c.running.Store(true)
	c.Refresh()

	<-stopCh

	c.running.Store(false)
}

// Refresh updates the Controller configuration
func (c Controller) Refresh() error {
	if !c.running.Load().(bool) {
		return nil
	}

	// Clears the current configuration
	err := c.pf.Clear()
	if err != nil {
		return err
	}

	pats, err := c.patInformer.Lister().PortAddressTranslations("").List(labels.Everything())
	if err != nil {
		return err
	}
	for _, pat := range pats {
		service, err := c.serviceInformer.Lister().Services(pat.Namespace).Get(pat.Spec.Service)
		if err != nil {
			fmt.Printf("Failed to fetch service %s/%s. Ignoring resource %s/%s: %s\n", pat.Namespace, pat.Spec.Service, pat.Namespace, pat.Name, err.Error())
			continue
		}
		if service.Spec.Type != v1.ServiceTypeClusterIP {
			fmt.Printf("Invalid service provided. %s/%s must be of type ClusterIP. Ignoring resource %s/%s\n", pat.Namespace, pat.Spec.Service, pat.Namespace, pat.Name)
			continue
		}
		fmt.Printf("Configuring service %s/%s\n", pat.Namespace, pat.Spec.Service)
		servicePort := service.Spec.Ports[0]
		err = c.pf.Forward(servicePort.Protocol, pat.Spec.Port, service.Spec.ClusterIP, servicePort.Port)
		if err != nil {
			fmt.Printf("Failed to setup forwarding. Ignoring resource %s/%s: %s\n", pat.Namespace, pat.Name, err.Error())
		}
	}

	c.UpdateLoadBalancer(pats)

	c.pf.Print()

	return nil
}

// UpdateLoadBalancer add ports to the load balancer
func (c Controller) UpdateLoadBalancer(pats []*patv1beta1.PortAddressTranslation) {
	udpNamespace, udpName := split(c.opt.UDPService)
	lbService, err := c.serviceInformer.Lister().Services(udpNamespace).Get(udpName)
	if err != nil {
		fmt.Println("Failed to fetch the service")
		return
	}
	var shouldSave = false
	configuredPort := map[int32]bool{}
	for i, port := range lbService.Spec.Ports {
		configuredPort[port.Port] = true
		if port.Name == "" {
			lbService.Spec.Ports[i].Name = fmt.Sprintf("%d", port.Port)
		}
	}
	for _, pat := range pats {
		service, err := c.serviceInformer.Lister().Services(pat.Namespace).Get(pat.Spec.Service)
		if err != nil || service.Spec.Ports[0].Protocol != v1.ProtocolUDP {
			continue
		}
		if !configuredPort[pat.Spec.Port] {
			configuredPort[pat.Spec.Port] = true
			shouldSave = true

			lbService.Spec.Ports = append(lbService.Spec.Ports, v1.ServicePort{
				Name:     fmt.Sprintf("%s-%s-%d", service.Namespace, service.Name, pat.Spec.Port),
				Protocol: v1.ProtocolUDP,
				Port:     pat.Spec.Port,
			})
		}
	}

	if !shouldSave {
		return
	}

	fmt.Println("Updating UDP load balancer")
	_, err = c.opt.KubeClientSet.CoreV1().Services(udpNamespace).Update(lbService)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	// service.Spec.
	// c.opt.TCPService
}

func split(s string) (namespace, name string) {
	parts := strings.SplitN(s, "/", 2)
	return parts[0], parts[1]
}
