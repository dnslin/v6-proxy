package netutils

import (
	"crypto/rand"
	"errors"
	"net"
)

func GetIPAddress(domain string) (string, bool, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return "", false, err
	}

	for _, ip := range ips {
		if ip.To4() == nil {
			return ip.String(), true, nil
		}
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), false, nil
		}
	}

	return "", false, net.InvalidAddrError("No valid IP addresses found")
}

func RandomV6(network string) (net.IP, error) {
	_, subnet, err := net.ParseCIDR(network)
	if err != nil {
		return nil, err
	}

	// Make sure we're dealing with an IPv6 network
	if subnet.IP.To4() != nil {
		return nil, errors.New("expected an IPv6 network")
	}

	// Create a new IP address buffer and copy the network prefix
	ip := make(net.IP, len(subnet.IP))
	copy(ip, subnet.IP)

	// Generate random bytes for the host part of the address
	hostBytes := make([]byte, len(ip)-len(subnet.Mask))
	_, err = rand.Read(hostBytes)
	if err != nil {
		return nil, err
	}

	// Apply the random bytes to the host part of the IP address, respecting the subnet mask
	for i, b := range hostBytes {
		ip[len(subnet.Mask)+i] = b
	}

	// Apply the mask to ensure the network part remains unchanged
	for i := 0; i < len(ip); i++ {
		ip[i] = ip[i] | ^subnet.Mask[i]
	}

	return ip, nil
}
