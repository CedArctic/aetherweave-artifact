package testing

import (
	"context"
	"crypto/ecdsa"
	"errors"

	ethpb "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// MockPeerManager is mock of the PeerManager interface.
type MockPeerManager struct {
	Enr               *enr.Record
	PID               peer.ID
	BHost             host.Host
	DiscoveryAddr     []multiaddr.Multiaddr
	FailDiscoveryAddr bool
}

// Disconnect .
func (*MockPeerManager) Disconnect(peer.ID) error {
	return nil
}

// PeerID .
func (m *MockPeerManager) PeerID() peer.ID {
	return m.PID
}

// Host .
func (m *MockPeerManager) Host() host.Host {
	return m.BHost
}

// ENR .
func (m *MockPeerManager) ENR() *enr.Record {
	return m.Enr
}

// DiscoveryAddresses .
func (m *MockPeerManager) DiscoveryAddresses() ([]multiaddr.Multiaddr, error) {
	if m.FailDiscoveryAddr {
		return nil, errors.New("fail")
	}
	return m.DiscoveryAddr, nil
}

// RefreshPersistentSubnets .
func (*MockPeerManager) RefreshPersistentSubnets() {}

// FindPeersWithSubnet .
func (*MockPeerManager) FindPeersWithSubnet(_ context.Context, _ string, _ uint64, _ int) (bool, error) {
	return true, nil
}

// AddPingMethod .
func (*MockPeerManager) AddPingMethod(_ func(ctx context.Context, id peer.ID) error) {}

func (s *MockPeerManager) LocalNetRec() *ethpb.NetworkRecord { return nil }

func (s *MockPeerManager) GetPrivKey() *ecdsa.PrivateKey { return nil }

func (s *MockPeerManager) StaticNetRecs() []*ethpb.NetworkRecord { return nil }

func (s *MockPeerManager) IsPeerAtLimit(inbound bool) bool { return false }

func (s *MockPeerManager) GetMaxPeers() uint { return 0 }
