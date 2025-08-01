// stun.go
package main

import (
	"errors"
	"net"

	"github.com/pion/stun"
)

// getPublicIP uses a STUN server to discover the public IP address and port.
func getPublicIP(stunServer string) (string, error) {
	// Create a new UDP connection to the STUN server.
	conn, err := net.Dial("udp", stunServer)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create a new STUN client.
	client, err := stun.NewClient(conn)
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a binding request.
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	var publicAddr string
	// The callback function will be called when a response is received.
	callback := func(res stun.Event) {
		if res.Error != nil {
			err = res.Error
			return
		}

		var xorAddr stun.XORMappedAddress
		if err = xorAddr.GetFrom(res.Message); err != nil {
			return
		}
		publicAddr = xorAddr.String()
	}

	// Send the request and wait for the response.
	if err = client.Do(message, callback); err != nil {
		return "", err
	}

	if publicAddr == "" {
		return "", errors.New("failed to get public IP from STUN server")
	}

	return publicAddr, nil
}
