package helpers

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

type ConfigRemote struct {
	Name     string
	Username string
	Hostname string
	Key      string
	KeyPass  string
}

// Connects to the remote server and returns a *ssh.Client
func (c *ConfigRemote) Connect() (*ssh.Client, error) {
	keyBytes, err := os.ReadFile(c.Key)

	if err != nil {
		return nil, fmt.Errorf("error reading key file: %w", err)
	}

	var key ssh.Signer
	if c.KeyPass != "" {
		key, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(c.KeyPass))
	} else {
		key, err = ssh.ParsePrivateKey(keyBytes)
	}

	if err != nil {
		return nil, fmt.Errorf("error reading key file: %w", err)
	}

	// Connect to the remote server
	client, err := ssh.Dial("tcp", c.Hostname, &ssh.ClientConfig{
		User: c.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		// TODO: Add host key checking
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}
