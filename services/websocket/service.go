package websocket

/*
 */

import (
	"net/http"

	"net"
	"strconv"

	"fmt"

	"time"

	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/sda"
	"golang.org/x/net/websocket"
)

// ServiceName is the name to refer to the Template service from another
// package.
const ServiceName = "WebSocket"

func init() {
	sda.RegisterNewService(ServiceName, newService)
}

// Service is our template-service
type Service struct {
	*sda.ServiceProcessor
	path string
}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
// If you use CreateProtocolSDA, this will not be called, as the SDA will
// instantiate the protocol on its own. If you need more control at the
// instantiation of the protocol, use CreateProtocolService, and you can
// give some extra-configuration to your protocol in here.
func (s *Service) NewProtocol(tn *sda.TreeNodeInstance, conf *sda.GenericConfig) (sda.ProtocolInstance, error) {
	log.Lvl3("Not templated yet")
	return nil, nil
}

func (s *Service) Shutdown() {
	log.Lvl1("Shutting down service websocket")
}

func (s *Service) rootHandler(ws *websocket.Conn) {
	buf := make([]byte, 256)
	for {
		n, err := ws.Read(buf)
		log.ErrFatal(err)
		log.Print(string(buf))
		if n == 256 {
			break
		}
	}
}

func getWebHost(si *network.ServerIdentity) (string, error) {
	log.Print(si.Addresses[0])
	host, portStr, err := net.SplitHostPort(si.Addresses[0])
	if err != nil {
		return "", err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, port+100), nil
}

// newTemplate receives the context and a path where it can write its
// configuration, if desired. As we don't know when the service will exit,
// we need to save the configuration on our own from time to time.
func newService(c *sda.Context, path string) sda.Service {
	s := &Service{
		ServiceProcessor: sda.NewServiceProcessor(c),
		path:             path,
	}

	http.Handle("/root", websocket.Handler(s.rootHandler))
	go func() {
		time.Sleep(1 * time.Second)
		webHost, err := getWebHost(c.ServerIdentity())
		log.ErrFatal(err)
		log.ErrFatal(http.ListenAndServe(webHost, nil))
	}()

	return s
}