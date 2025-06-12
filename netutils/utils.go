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
		return nil, errors.New("expected an IPv6 network, but got an IPv4 network")
	}

	// Generate 16 random bytes for a full IPv6 address
	randomBytes := make([]byte, 16)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// Create a new IP address by combining the network prefix with the random host part.
	randomIP := make(net.IP, 16)
	for i := 0; i < 16; i++ {
		// Keep the network prefix part
		prefixPart := subnet.IP[i] & subnet.Mask[i]
		// Get the host part from the random bytes
		hostPart := randomBytes[i] & ^subnet.Mask[i]
		// Combine into the new IP byte
		randomIP[i] = prefixPart | hostPart
	}

	return randomIP, nil
}
