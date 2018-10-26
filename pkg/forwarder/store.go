package forwarder

import (
	"fmt"
	"os"

	patv1beta1 "github.com/pdeslaur/kube-pat/pkg/apis/portaddresstranslation/v1beta1"
	informers "github.com/pdeslaur/kube-pat/pkg/client/informers/externalversions/portaddresstranslation/v1beta1"
	listers "github.com/pdeslaur/kube-pat/pkg/client/listers/portaddresstranslation/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1informers "k8s.io/client-go/informers/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// PortForwardingConfig is a requested port forwarding operation.
type PortForwardingConfig struct {
	Protocol                   corev1.Protocol
	SrcPort                    int32
	DestIP                     string
	DestPort                   int32
	PortAddressTranslationName string
	ServiceName                string
}

// Store is offering interfaces for intercting with cached entities.
type Store struct {
	patLister     listers.PortAddressTranslationLister
	serviceLister corev1listers.ServiceLister
}

// NewStore creates a new store.
func NewStore(
	patInformer informers.PortAddressTranslationInformer,
	serviceInformer corev1informers.ServiceInformer,
	refreshFunc func(store *Store) error,
) *Store {
	s := new(Store)
	s.patLister = patInformer.Lister()
	s.serviceLister = serviceInformer.Lister()

	patInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(interface{}) {
				refreshFunc(s)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if oldObj.(*patv1beta1.PortAddressTranslation).GetResourceVersion() != newObj.(*patv1beta1.PortAddressTranslation).GetResourceVersion() {
					refreshFunc(s)
				}
			},
			DeleteFunc: func(interface{}) {
				refreshFunc(s)
			},
		})

	serviceInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(interface{}) {
				refreshFunc(s)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if oldObj.(*corev1.Service).GetResourceVersion() != newObj.(*corev1.Service).GetResourceVersion() {
					refreshFunc(s)
				}
			},
			DeleteFunc: func(interface{}) {
				refreshFunc(s)
			},
		})

	return s
}

func (s Store) createFromPat(pat *patv1beta1.PortAddressTranslation) (PortForwardingConfig, error) {
	service, err := s.serviceLister.Services(pat.Namespace).Get(pat.Spec.Service)
	if err != nil {
		return PortForwardingConfig{}, fmt.Errorf("failed to fetch service %s/%s: %s", pat.Namespace, pat.Spec.Service, err.Error())
	}
	if service.Spec.Type != corev1.ServiceTypeClusterIP {
		return PortForwardingConfig{}, fmt.Errorf("service %s/%s must be of type ClusterIP to be compatible with %s/%s", pat.Namespace, pat.Spec.Service, pat.Namespace, pat.Name)
	}

	return PortForwardingConfig{
		Protocol:                   service.Spec.Ports[0].Protocol,
		SrcPort:                    pat.Spec.Port,
		DestIP:                     service.Spec.ClusterIP,
		DestPort:                   service.Spec.Ports[0].Port,
		PortAddressTranslationName: fmt.Sprintf("%s/%s", pat.Namespace, pat.Name),
		ServiceName:                fmt.Sprintf("%s/%s", service.Namespace, service.Name),
	}, nil
}

// Iterate walks through all PortForwardingConfig
func (s Store) Iterate() <-chan PortForwardingConfig {
	chnl := make(chan PortForwardingConfig)
	go func() {
		pats, err := s.patLister.PortAddressTranslations("").List(labels.Everything())
		if err != nil {
			panic(err)
		}
		for _, pat := range pats {
			pfc, err := s.createFromPat(pat)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				continue
			}
			chnl <- pfc
		}
		close(chnl)
	}()
	return chnl
}

func withoutArgs(f func() error) func(interface{}) {
	return func(interface{}) {
		f()
	}
}
