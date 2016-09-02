package network

import (
	"errors"
	"sync"
)

// Dispatcher is an interface whose sole role is to distribute messages to the
// right Processor. No processing is done,i.e. no looking at packet content.
// There are many ways to distribute messages: for the moment, only
// BlockingDispatcher is implemented, which is a blocking dispatcher.
// Later, one can easily imagine to have a dispatcher with one worker in a
// goroutine or a fully fledged producer/consumers pattern in go routines.
// Each Processor that wants to receive all messages of a specific
// type must register itself to the dispatcher using `RegisterProcessor()`.
// The network layer must call `Dispatch()` each time it receives a message, so
// the dispatcher is able to dispatch correctly to the right Processor for
// further analysis.
type Dispatcher interface {
	// RegisterProcessor is called by a Processor so it can receive all packets
	// of type msgType. If given multiple msgType, the same processor will be
	// called for each of all the msgType given.
	// **NOTE** In the current version, if a subequent call to RegisterProcessor
	// happens, for the same msgType, the latest Processor will be used; there
	// is no *copy* or *duplication* of messages.
	RegisterProcessor(p Processor, msgType ...PacketTypeID)
	// RegisterProcessorFunc enables to register directly a function that will
	// be called for each packet of type msgType. It's a shorter way of
	// registering a Processor.
	RegisterProcessorFunc(PacketTypeID, func(*Packet))
	// Dispatch will find the right processor to dispatch the packet to. The id
	// is the identity of the author / sender of the packet.
	// It can be called for example by the network layer.
	// If no processor is found for this message type, then an error is returned
	Dispatch(packet *Packet) error
}

// Processor is an abstraction to represent any object that want to process
// packets. It is used in conjunction with Dispatcher:
// A processor must register itself to a Dispatcher so the Dispatcher will
// dispatch every messages to the Processor asked for.
type Processor interface {
	// Process takes a received Packet.
	Process(packet *Packet)
}

// BlockingDispatcher is a Dispatcher that simply calls `p.Process()` on a
// processor p each time it receives a message with `Dispatch`. It does *not*
// launch a go routine, or put the message in a queue, etc.
// It can be re-used for more complex dispatcher.
type BlockingDispatcher struct {
	sync.Mutex
	procs map[PacketTypeID]Processor
}

// NewBlockingDispatcher will return a freshly initialized BlockingDispatcher.
func NewBlockingDispatcher() *BlockingDispatcher {
	return &BlockingDispatcher{
		procs: make(map[PacketTypeID]Processor),
	}
}

// RegisterProcessor saves the given processor in the dispatcher.
func (d *BlockingDispatcher) RegisterProcessor(p Processor, msgType ...PacketTypeID) {
	d.Lock()
	defer d.Unlock()
	for _, t := range msgType {
		d.procs[t] = p
	}
}

// RegisterProcessorFunc takes a func, creates a Processor struct around it and
// register it to the dispatcher. It's a more straightforward way to register a
// Processor.
func (d *BlockingDispatcher) RegisterProcessorFunc(msgType PacketTypeID, fn func(*Packet)) {
	p := &defaultProcessor{
		fn: fn,
	}
	d.RegisterProcessor(p, msgType)
}

// Dispatch will directly call the right processor's method Process. It's a
// blocking call if the Processor is blocking !
func (d *BlockingDispatcher) Dispatch(packet *Packet) error {
	d.Lock()
	var p Processor
	if p = d.procs[packet.MsgType]; p == nil {
		d.Unlock()
		return errors.New("No Processor attached to this message type " + packet.MsgType.String())
	}
	d.Unlock()
	p.Process(packet)
	return nil
}

// RoutineDispatcher is a Dispatcher that will dispatch messages to Processor
// in a go routine. RoutineDispatcher creates one go routine per messages it
// receives.
type RoutineDispatcher struct {
	*BlockingDispatcher
}

// NewRoutineDispatcher returns a fresh RoutineDispatcher
func NewRoutineDispatcher() *RoutineDispatcher {
	return &RoutineDispatcher{
		BlockingDispatcher: NewBlockingDispatcher(),
	}
}

// Dispatch implements the Dispatcher interface. It will give the packet to the
// right Processor in a go routine.
func (d *RoutineDispatcher) Dispatch(packet *Packet) error {
	var p = d.procs[packet.MsgType]
	if p == nil {
		return errors.New("No Processor attached to this message type.")
	}
	go p.Process(packet)
	return nil
}

type defaultProcessor struct {
	fn func(*Packet)
}

func (dp *defaultProcessor) Process(msg *Packet) {
	dp.fn(msg)
}