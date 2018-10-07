package forwarder

import (
	"errors"
	"fmt"
	"os"

	"github.com/coreos/go-iptables/iptables"
	"k8s.io/api/core/v1"
)

const (
	nic = "eth0"
)

type portMap map[int32]bool

// PortForwarder configures port address translation to redirect L3 traffic.
type PortForwarder struct {
	ipt   *iptables.IPTables
	ports map[v1.Protocol]portMap
}

// NewPortForwarder creates a new PortForwarder.
func NewPortForwarder() (*PortForwarder, error) {
	ipt, err := iptables.New()
	if err != nil {
		return nil, err
	}

	err = ipt.Append("nat", "POSTROUTING", "-o", nic, "-j", "MASQUERADE")
	if err != nil {
		return nil, errors.New("Failed to configure IPTables with postrouting masquerade")
	}

	pf := new(PortForwarder)
	pf.ipt = ipt
	pf.ports = map[v1.Protocol]portMap{
		v1.ProtocolUDP: portMap{},
		v1.ProtocolTCP: portMap{},
	}
	return pf, nil
}

// NewPortForwarderOrDie creates a new PortForwarder or dies.
func NewPortForwarderOrDie() *PortForwarder {
	pf, err := NewPortForwarder()
	if err != nil {
		panic(nil)
	}
	return pf
}

// Clear clears the current forwarding configuration.
func (pf PortForwarder) Clear() error {
	pf.resetPorts()
	return pf.ipt.ClearChain("nat", "PREROUTING")
}

// Forward configures a new forwarding rule.
func (pf PortForwarder) Forward(p v1.Protocol, srcPort int32, destIP string, destPort int32) error {
	if err := pf.registerPort(p, srcPort); err != nil {
		return err
	}
	destination := fmt.Sprintf("%s:%d", destIP, destPort)
	return pf.ipt.Append("nat", "PREROUTING", "-p", string(p), "-i", nic, "--dport", fmt.Sprint(srcPort), "-j", "DNAT", "--to-destination", destination)
}

// Print prints the IPTables rules of the PREROUTING table.
func (pf PortForwarder) Print() {
	fmt.Println("IPTables PREROUTING configuration:")
	rules, err := pf.ipt.List("nat", "PREROUTING")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch IPTables rules: %s\n", err.Error())
		return
	}
	for _, rule := range rules {
		fmt.Println(rule)
	}
}

func (pf PortForwarder) resetPorts() {
	pf.ports[v1.ProtocolUDP] = portMap{}
	pf.ports[v1.ProtocolTCP] = portMap{}
}

func (pf PortForwarder) registerPort(protocol v1.Protocol, port int32) error {
	if pf.ports[protocol][port] {
		return fmt.Errorf("Port %s:%d is already taken", protocol, port)
	}
	pf.ports[protocol][port] = true
	return nil
}
