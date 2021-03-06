package virtualbox

import (
	"net"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

const stdOutOneHostOnlyNetwork = `Name:            vboxnet0
GUID:            786f6276-656e-4074-8000-0a0027000000
DHCP:            Disabled
IPAddress:       192.168.99.1
NetworkMask:     255.255.255.0
IPV6Address:
IPV6NetworkMaskPrefixLength: 0
HardwareAddress: 0a:00:27:00:00:00
MediumType:      Ethernet
Status:          Up
VBoxNetworkName: HostInterfaceNetworking-vboxnet0

`
const stdOutTwoHostOnlyNetwork = `Name:            vboxnet0
GUID:            786f6276-656e-4074-8000-0a0027000000
DHCP:            Disabled
IPAddress:       192.168.99.1
NetworkMask:     255.255.255.0
IPV6Address:
IPV6NetworkMaskPrefixLength: 0
HardwareAddress: 0a:00:27:00:00:00
MediumType:      Ethernet
Status:          Up
VBoxNetworkName: HostInterfaceNetworking-vboxnet0

Name:            vboxnet1
GUID:            786f6276-656e-4174-8000-0a0027000001
DHCP:            Disabled
IPAddress:       192.168.99.1
NetworkMask:     255.255.255.0
IPV6Address:
IPV6NetworkMaskPrefixLength: 0
HardwareAddress: 0a:00:27:00:00:01
MediumType:      Ethernet
Status:          Up
VBoxNetworkName: HostInterfaceNetworking-vboxnet1

`

// Tests that when we have a host only network which matches our expectations,
// it gets returned correctly.
func TestGetHostOnlyNetworkHappy(t *testing.T) {
	cidr := "192.168.99.0/24"
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("Error parsing cidr: %s", err)
	}
	expectedHostOnlyNetwork := &hostOnlyNetwork{
		IPv4: *ipnet,
	}
	vboxNets := map[string]*hostOnlyNetwork{
		"HostInterfaceNetworking-vboxnet0": expectedHostOnlyNetwork,
	}

	n := getHostOnlyNetwork(vboxNets, ip, ipnet.Mask)
	if !reflect.DeepEqual(n, expectedHostOnlyNetwork) {
		t.Fatalf("Expected result of calling getHostOnlyNetwork to be the same as expected but it was not:\nexpected: %+v\nactual: %+v\n", expectedHostOnlyNetwork, n)
	}
}

// Tests that we are able to properly detect when a host only network which
// matches our expectations can not be found.
func TestGetHostOnlyNetworkNotFound(t *testing.T) {
	// Note that this has a different ip is different from "ip" below.
	cidr := "192.168.99.0/24"
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("Error parsing cidr: %s", err)
	}

	ip = net.ParseIP("192.168.59.0").To4()

	// Suppose a vbox net is created, but it doesn't align with our
	// expectation.
	vboxNet := &hostOnlyNetwork{
		IPv4: *ipnet,
	}
	vboxNets := map[string]*hostOnlyNetwork{
		"HostInterfaceNetworking-vboxnet0": vboxNet,
	}

	n := getHostOnlyNetwork(vboxNets, ip, ipnet.Mask)
	if n != nil {
		t.Fatalf("Expected vbox net to be nil but it has a value: %+v\n", n)
	}
}

// Tests a special case where Virtualbox creates the host only network
// successfully but mis-reports the netmask.
func TestGetHostOnlyNetworkWindows10Bug(t *testing.T) {
	cidr := "192.168.99.0/24"
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("Error parsing cidr: %s", err)
	}

	// This is a faulty netmask: a VirtualBox bug causes it to be
	// misreported.
	ipnet.Mask = net.IPMask(net.ParseIP("15.0.0.0").To4())

	expectedHostOnlyNetwork := &hostOnlyNetwork{
		IPv4: *ipnet,
	}

	vboxNets := map[string]*hostOnlyNetwork{
		"HostInterfaceNetworking-vboxnet0": expectedHostOnlyNetwork,
	}

	// The Mask that we are passing in will be the "legitimate" mask, so it
	// must differ from the magic buggy mask.
	n := getHostOnlyNetwork(vboxNets, ip, net.IPMask(net.ParseIP("255.255.255.0").To4()))
	if !reflect.DeepEqual(n, expectedHostOnlyNetwork) {
		t.Fatalf("Expected result of calling getHostOnlyNetwork to be the same as expected but it was not:\nexpected: %+v\nactual: %+v\n", expectedHostOnlyNetwork, n)
	}
}

func TestListHostOnlyNetworks(t *testing.T) {
	vbox := &VBoxManagerMock{
		args:   "list hostonlyifs",
		stdOut: stdOutOneHostOnlyNetwork,
	}

	nets, err := listHostOnlyNetworks(vbox)

	assert.Equal(t, 1, len(nets))
	assert.NoError(t, err)

	net, present := nets["HostInterfaceNetworking-vboxnet0"]

	assert.True(t, present)
	assert.Equal(t, "vboxnet0", net.Name)
	assert.Equal(t, "786f6276-656e-4074-8000-0a0027000000", net.GUID)
	assert.False(t, net.DHCP)
	assert.Equal(t, "192.168.99.1", net.IPv4.IP.String())
	assert.Equal(t, "ffffff00", net.IPv4.Mask.String())
	assert.Empty(t, net.IPv6.IP)
	assert.Equal(t, "0a:00:27:00:00:00", net.HwAddr.String())
	assert.Equal(t, "Ethernet", net.Medium)
	assert.Equal(t, "Up", net.Status)
	assert.Equal(t, "HostInterfaceNetworking-vboxnet0", net.NetworkName)
}

func TestListTwoHostOnlyNetworks(t *testing.T) {
	vbox := &VBoxManagerMock{
		args:   "list hostonlyifs",
		stdOut: stdOutTwoHostOnlyNetwork,
	}

	nets, err := listHostOnlyNetworks(vbox)

	assert.Equal(t, 2, len(nets))
	assert.NoError(t, err)

	net, present := nets["HostInterfaceNetworking-vboxnet1"]

	assert.True(t, present)
	assert.Equal(t, "vboxnet1", net.Name)
	assert.Equal(t, "786f6276-656e-4174-8000-0a0027000001", net.GUID)
	assert.False(t, net.DHCP)
	assert.Equal(t, "192.168.99.1", net.IPv4.IP.String())
	assert.Equal(t, "ffffff00", net.IPv4.Mask.String())
	assert.Empty(t, net.IPv6.IP)
	assert.Equal(t, "0a:00:27:00:00:01", net.HwAddr.String())
	assert.Equal(t, "Ethernet", net.Medium)
	assert.Equal(t, "Up", net.Status)
	assert.Equal(t, "HostInterfaceNetworking-vboxnet1", net.NetworkName)
}

func TestListHostOnlyNetworksDontRelyOnEmptyLinesForParsing(t *testing.T) {
	vbox := &VBoxManagerMock{
		args: "list hostonlyifs",
		stdOut: `Name:            vboxnet0
VBoxNetworkName: HostInterfaceNetworking-vboxnet0
Name:            vboxnet1
VBoxNetworkName: HostInterfaceNetworking-vboxnet1`,
	}

	nets, err := listHostOnlyNetworks(vbox)

	assert.Equal(t, 2, len(nets))
	assert.NoError(t, err)

	net, present := nets["HostInterfaceNetworking-vboxnet1"]
	assert.True(t, present)
	assert.Equal(t, "vboxnet1", net.Name)

	net, present = nets["HostInterfaceNetworking-vboxnet0"]
	assert.True(t, present)
	assert.Equal(t, "vboxnet0", net.Name)
}

func TestGetHostOnlyNetwork(t *testing.T) {
	vbox := &VBoxManagerMock{
		args:   "list hostonlyifs",
		stdOut: stdOutOneHostOnlyNetwork,
	}

	net, err := getOrCreateHostOnlyNetwork(net.ParseIP("192.168.99.1"), parseIPv4Mask("255.255.255.0"), nil, nil, nil, vbox)

	assert.NotNil(t, net)
	assert.Equal(t, "HostInterfaceNetworking-vboxnet0", net.NetworkName)
	assert.NoError(t, err)
}

func TestFailWithDuplicateHostOnlyNetworks(t *testing.T) {
	vbox := &VBoxManagerMock{
		args:   "list hostonlyifs",
		stdOut: stdOutTwoHostOnlyNetwork,
	}

	net, err := getOrCreateHostOnlyNetwork(net.ParseIP("192.168.99.1"), parseIPv4Mask("255.255.255.0"), nil, nil, nil, vbox)

	assert.Nil(t, net)
	assert.Equal(t, errDuplicateHostOnlyInterfaceNetworks, err)
}
