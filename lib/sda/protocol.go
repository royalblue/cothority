package sda

import (
	"errors"
)

// NewProtocol is the function-signature needed to instantiate a new protocol
type NewProtocol func(*Host, *Tree) ProtocolInstance

// protocols holds a map of all available protocols and how to create an
// instance of it
var protocols map[string]NewProtocol

// Protocol is the interface that instances have to use in order to be
// recognized as protocols
type ProtocolInstance interface {
	// Dispatch is called whenever packets are ready and should be treated
	Dispatch(m *SDAData) error
}

// ProtocolInstantiate creates a new instance of a protocol given by it's name
func ProtocolInstantiate(protoName string, n *Host, t *Tree) (ProtocolInstance, error) {
	p, ok := protocols[protoName]
	if !ok {
		return nil, errors.New("Protocol doesn't exist")
	}
	return p(n, t), nil
}

// ProtocolRegister takes a protocol and registers it under a given name.
// As this might be called from an 'init'-function, we need to check the
// initialisation of protocols here and not in our own 'init'.
func ProtocolRegister(protoName string, protocol NewProtocol) {
	if protocols == nil {
		protocols = make(map[string]NewProtocol)
	}
	protocols[protoName] = protocol
}

// ProtocolExists returns whether a certain protocol already has been
// registered
func ProtocolExists(protoName string) bool {
	_, ok := protocols[protoName]
	return ok
}
