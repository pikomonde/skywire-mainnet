package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/dmsg"
	"github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/dmsg/disc"
)

// Default ports.
// TODO(evanlinjin): Define these properly. These are currently random.
const (
	SetupPort      = uint16(36)  // Listening port of a setup node.
	AwaitSetupPort = uint16(136) // Listening port of a visor node for setup operations.
	TransportPort  = uint16(45)  // Listening port of a visor node for incoming transports.
)

// Networks.
const (
	DmsgNet = "dmsg"
)

var (
	ErrUnknownNetwork = errors.New("unknown network type")
)

type Config struct {
	PubKey     cipher.PubKey
	SecKey     cipher.SecKey
	TpNetworks []string // networks to be used with transports

	DmsgDiscAddr string
	DmsgMinSrvs  int
}

// Network represents
type Network struct {
	conf  Config
	dmsgC *dmsg.Client
}

func New(conf Config) *Network {
	dmsgC := dmsg.NewClient(conf.PubKey, conf.SecKey, disc.NewHTTP(conf.DmsgDiscAddr), dmsg.SetLogger(logging.MustGetLogger("network.dmsgC")))
	return &Network{
		conf:  conf,
		dmsgC: dmsgC,
	}
}

func (n *Network) Init(ctx context.Context) error {
	fmt.Println("dmsg: min_servers:", n.conf.DmsgMinSrvs)
	if err := n.dmsgC.InitiateServerConnections(ctx, n.conf.DmsgMinSrvs); err != nil {
		return fmt.Errorf("failed to initiate 'dmsg': %v", err)
	}
	return nil
}

func (n *Network) Close() error {
	wg := new(sync.WaitGroup)
	wg.Add(1)

	var dmsgErr error
	go func() {
		dmsgErr = n.dmsgC.Close()
		wg.Done()
	}()

	wg.Wait()
	if dmsgErr != nil {
		return dmsgErr
	}
	return nil
}

func (n *Network) LocalPK() cipher.PubKey { return n.conf.PubKey }

func (n *Network) LocalSK() cipher.SecKey { return n.conf.SecKey }

func (n *Network) Dmsg() *dmsg.Client { return n.dmsgC }

func (n *Network) Dial(network string, pk cipher.PubKey, port uint16) (*Conn, error) {
	ctx := context.Background()
	switch network {
	case DmsgNet:
		conn, err := n.dmsgC.Dial(ctx, pk, port)
		if err != nil {
			return nil, err
		}
		return makeConn(conn, network), nil
	default:
		return nil, ErrUnknownNetwork
	}
}

func (n *Network) Listen(network string, port uint16) (*Listener, error) {
	switch network {
	case DmsgNet:
		lis, err := n.dmsgC.Listen(port)
		if err != nil {
			return nil, err
		}
		return makeListener(lis, network), nil
	default:
		return nil, ErrUnknownNetwork
	}
}

type Listener struct {
	net.Listener
	lPK     cipher.PubKey
	lPort   uint16
	network string
}

func makeListener(l net.Listener, network string) *Listener {
	lPK, lPort := disassembleAddr(l.Addr())
	return &Listener{Listener: l, lPK: lPK, lPort: lPort, network: network}
}

func (l Listener) LocalPK() cipher.PubKey { return l.lPK }
func (l Listener) LocalPort() uint16      { return l.lPort }
func (l Listener) Network() string        { return l.network }

func (l Listener) AcceptConn() (*Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return makeConn(conn, l.network), nil
}

type Conn struct {
	net.Conn
	lPK     cipher.PubKey
	rPK     cipher.PubKey
	lPort   uint16
	rPort   uint16
	network string
}

func makeConn(conn net.Conn, network string) *Conn {
	lPK, lPort := disassembleAddr(conn.LocalAddr())
	rPK, rPort := disassembleAddr(conn.RemoteAddr())
	return &Conn{Conn: conn, lPK: lPK, rPK: rPK, lPort: lPort, rPort: rPort, network: network}
}

func (c Conn) LocalPK() cipher.PubKey  { return c.lPK }
func (c Conn) RemotePK() cipher.PubKey { return c.rPK }
func (c Conn) LocalPort() uint16       { return c.lPort }
func (c Conn) RemotePort() uint16      { return c.rPort }
func (c Conn) Network() string         { return c.network }

func disassembleAddr(addr net.Addr) (pk cipher.PubKey, port uint16) {
	strs := strings.Split(addr.String(), ":")
	if len(strs) != 2 {
		panic(fmt.Errorf("network.disassembleAddr: %v %s", "invalid addr", addr.String()))
	}
	if err := pk.Set(strs[0]); err != nil {
		panic(fmt.Errorf("network.disassembleAddr: %v %s", err, addr.String()))
	}
	if strs[1] != "~" {
		if _, err := fmt.Sscanf(strs[1], "%d", &port); err != nil {
			panic(fmt.Errorf("network.disassembleAddr: %v", err))
		}
	}
	return
}