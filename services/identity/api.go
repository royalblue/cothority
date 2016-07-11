package identity

import (
	"errors"
	"io"

	"io/ioutil"

	"github.com/dedis/cothority/crypto"
	"github.com/dedis/cothority/log"
	"github.com/dedis/cothority/network"
	"github.com/dedis/cothority/sda"
	"github.com/dedis/cothority/services/skipchain"
	"github.com/dedis/crypto/abstract"
	"github.com/dedis/crypto/config"
)

/*
 */

func init() {
	for _, s := range []interface{}{
		&Device{},
		&Identity{},
		&Config{},
		&AddIdentity{},
		&AddIdentityReply{},
		&PropagateIdentity{},
		&ProposeSend{},
		&AttachToIdentity{},
		&ProposeFetch{},
		&ConfigUpdate{},
		&UpdateSkipBlock{},
		&ProposeVote{},
	} {
		network.RegisterMessageType(s)
	}
}

// Identity can both follow and update an IdRoster
type Identity struct {
	*sda.Client
	Private    abstract.Scalar
	Public     abstract.Point
	ID         ID
	Config     *Config
	Proposed   *Config
	ManagerStr string
	Cothority  *sda.Roster
	skipchain  *skipchain.Client
	root       *skipchain.SkipBlock
	data       *skipchain.SkipBlock
}

// NewIdentity starts a new identity that can contain multiple managers with
// different accounts
func NewIdentity(cothority *sda.Roster, majority int, owner string) *Identity {
	client := sda.NewClient(ServiceName)
	kp := config.NewKeyPair(network.Suite)
	return &Identity{
		Client:     client,
		Private:    kp.Secret,
		Public:     kp.Public,
		Config:     NewConfig(majority, kp.Public, owner),
		ManagerStr: owner,
		Cothority:  cothority,
		skipchain:  skipchain.NewClient(),
	}
}

// NewIdentityFromCothority searches for a given cothority
func NewIdentityFromCothority(el *sda.Roster, id ID) (*Identity, error) {
	iden := &Identity{
		Client:    sda.NewClient(ServiceName),
		Cothority: el,
		ID:        id,
	}
	err := iden.ConfigUpdate()
	if err != nil {
		return nil, err
	}
	return iden, nil
}

// NewIdentityFromStream reads the configuration of that client from
// any stream
func NewIdentityFromStream(in io.Reader) (*Identity, error) {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}
	_, id, err := network.UnmarshalRegistered(data)
	if err != nil {
		return nil, err
	}
	return id.(*Identity), nil
}

// SaveToStream stores the configuration of the client to a stream
func (i *Identity) SaveToStream(out io.Writer) error {
	data, err := network.MarshalRegisteredType(i)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}

// GetProposed returns the Propose-field or a copy of the config if
// the Propose-field is nil
func (i *Identity) GetProposed() *Config {
	if i.Proposed != nil {
		return i.Proposed
	}
	return i.Config.Copy()
}

// AttachToIdentity proposes to attach it to an existing Identity
func (i *Identity) AttachToIdentity(ID ID) error {
	i.ID = ID
	err := i.ConfigUpdate()
	if err != nil {
		return err
	}
	if _, exists := i.Config.Device[i.ManagerStr]; exists {
		return errors.New("Adding with an existing account-name")
	}
	confPropose := i.Config.Copy()
	confPropose.Device[i.ManagerStr] = &Device{i.Public}
	err = i.ProposeSend(confPropose)
	if err != nil {
		return err
	}
	return nil
}

// CreateIdentity asks the identityService to create a new Identity
func (i *Identity) CreateIdentity() error {
	msg, err := i.Send(i.Cothority.GetRandom(), &AddIdentity{i.Config, i.Cothority})
	if err != nil {
		return err
	}
	air := msg.Msg.(AddIdentityReply)
	i.root = air.Root
	i.data = air.Data
	i.ID = ID(i.data.Hash)

	return nil
}

// ProposeSend sends the new proposition of this identity
// ProposeVote
func (i *Identity) ProposeSend(il *Config) error {
	_, err := i.Send(i.Cothority.GetRandom(), &ProposeSend{i.ID, il})
	i.Proposed = il
	return err
}

// ProposeFetch verifies if there is a new configuration awaiting that
// needs approval from clients
func (i *Identity) ProposeFetch() error {
	msg, err := i.Send(i.Cothority.GetRandom(), &ProposeFetch{
		ID:          i.ID,
		AccountList: nil,
	})
	if err != nil {
		return err
	}
	cnc := msg.Msg.(ProposeFetch)
	i.Proposed = cnc.AccountList
	return nil
}

// ProposeVote calls the 'accept'-vote on the current propose-configuration
func (i *Identity) ProposeVote(accept bool) error {
	if i.Proposed == nil {
		return errors.New("No proposed config")
	}
	log.Lvlf3("Voting %t on %s", accept, i.Proposed.Device)
	if !accept {
		return nil
	}
	hash, err := i.Proposed.Hash()
	if err != nil {
		return err
	}
	sig, err := crypto.SignSchnorr(network.Suite, i.Private, hash)
	if err != nil {
		return err
	}
	msg, err := i.Send(i.Cothority.GetRandom(), &ProposeVote{
		ID:        i.ID,
		Signer:    i.ManagerStr,
		Signature: &sig,
	})
	err = sda.ErrMsg(msg, err)
	if err != nil {
		return err
	}
	sb, ok := msg.Msg.(skipchain.SkipBlock)
	if ok {
		log.Lvl2("Threshold reached and signed")
		i.data = &sb
		i.Config = i.Proposed
		i.Proposed = nil
	} else {
		log.Lvl2("Threshold not reached")
	}
	return nil
}

// ConfigUpdate asks if there is any new config available that has already
// been approved by others and updates the local configuration
func (i *Identity) ConfigUpdate() error {
	if i.Cothority == nil || len(i.Cothority.List) == 0 {
		return errors.New("Didn't find any list in the cothority")
	}
	msg, err := i.Send(i.Cothority.GetRandom(), &ConfigUpdate{ID: i.ID})
	if err != nil {
		return err
	}
	cu := msg.Msg.(ConfigUpdate)
	// TODO - verify new config
	i.Config = cu.AccountList
	return nil
}