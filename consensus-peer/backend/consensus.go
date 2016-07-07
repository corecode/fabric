/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backend

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/transport"

	"golang.org/x/net/context"

	"github.com/hyperledger/fabric/consensus"
	"github.com/hyperledger/fabric/consensus-peer/connection"
	"github.com/hyperledger/fabric/consensus-peer/persist"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("backend")

type Backend struct {
	consensus consensus.Consenter
	persist   *persist.Persist
	conn      *connection.Manager

	lock  sync.Mutex
	peers map[pb.PeerID]chan<- *pb.Message

	self     *PeerInfo
	peerInfo map[string]*PeerInfo
}

type consensusConn Backend

type PeerInfo struct {
	info connection.PeerInfo
	id   pb.PeerID
}

type peerInfoSlice []*PeerInfo

func (pi peerInfoSlice) Len() int {
	return len(pi)
}

func (pi peerInfoSlice) Less(i, j int) bool {
	return strings.Compare(pi[i].info.Fingerprint(), pi[j].info.Fingerprint()) == -1
}

func (pi peerInfoSlice) Swap(i, j int) {
	pi[i], pi[j] = pi[j], pi[i]
}

func New(persist *persist.Persist, conn *connection.Manager) (*Backend, error) {
	c := &Backend{
		conn:     conn,
		persist:  persist,
		peers:    make(map[pb.PeerID]chan<- *pb.Message),
		peerInfo: make(map[string]*PeerInfo),
	}

	prefix := "config.peers."
	peers, err := c.persist.ReadStateSet(prefix)
	if err != nil {
		return nil, err
	}

	var peerInfo []*PeerInfo
	for key, cert := range peers {
		addr := key[len(prefix):]

		pi, err := connection.NewPeerInfo(addr, cert)
		if err != nil {
			return nil, err
		}
		cpi := &PeerInfo{info: pi}
		if pi.Fingerprint() == conn.Self.Fingerprint() {
			c.self = cpi
		}
		peerInfo = append(peerInfo, cpi)
		c.peerInfo[pi.Fingerprint()] = cpi
	}

	sort.Sort(peerInfoSlice(peerInfo))
	for i, pi := range peerInfo {
		pi.id.Name = fmt.Sprintf("vp%d", i)
		logger.Infof("replica %d: %s", i, pi.info.Fingerprint())
	}

	if c.self == nil {
		return nil, fmt.Errorf("peer list does not contain local node")
	}

	logger.Infof("we are replica %s (%s)", c.self.id.Name, c.self.info)

	for _, peer := range c.peerInfo {
		if peer == c.self {
			continue
		}
		go c.connectWorker(peer)
	}
	RegisterConsensusServer(conn.Server, (*consensusConn)(c))
	return c, nil
}

func (c *Backend) RegisterConsenter(consensus consensus.Consenter) {
	c.consensus = consensus
}

func (c *Backend) connectWorker(peer *PeerInfo) {
	timeout := 1 * time.Second

	delay := time.After(0)
	for {
		// pace reconnect attempts
		<-delay

		// set up for next
		delay = time.After(timeout)

		logger.Infof("connecting to replica %s (%s)", peer.id.Name, peer.info)
		conn, err := c.conn.DialPeer(peer.info, grpc.WithBlock(), grpc.WithTimeout(timeout))
		if err != nil {
			logger.Warningf("could not connect to replica %s (%s): %s", peer.id.Name, peer.info, err)
			continue
		}

		ctx := context.TODO()

		client := NewConsensusClient(conn)
		consensus, err := client.Consensus(ctx, &Handshake{})
		if err != nil {
			logger.Warningf("could not establish consensus stream with replica %s (%s): %s", peer.id.Name, peer.info, err)
			continue
		}

		logger.Noticef("connection to replica %s (%s) established", peer.id.Name, peer.info)

		for {
			msg, err := consensus.Recv()
			if err == io.EOF || err == transport.ErrConnClosing {
				break
			}
			if err != nil {
				logger.Warningf("consensus stream with replica %s (%s) broke: %v", peer.id.Name, peer.info, err)
				break
			}
			c.consensus.RecvMsg(msg, &peer.id)
		}
	}
}

// gRPC interface
func (c *consensusConn) Consensus(_ *Handshake, srv Consensus_ConsensusServer) error {
	pi := connection.GetPeer(srv)
	peer, ok := c.peerInfo[pi.Fingerprint()]

	if !ok || !peer.info.Cert().Equal(pi.Cert()) {
		logger.Infof("rejecting connection from unknown replica %s", pi)
		return fmt.Errorf("unknown peer certificate")
	}
	logger.Infof("connection from replica %s (%s)", peer.id.Name, pi)

	ch := make(chan *pb.Message)
	c.lock.Lock()
	if oldch, ok := c.peers[peer.id]; ok {
		logger.Debugf("replacing connection from replica %s", peer.id.Name)
		close(oldch)
	}
	c.peers[peer.id] = ch
	c.lock.Unlock()

	var err error
	for msg := range ch {
		err = srv.Send(msg)
		if err != nil {
			c.lock.Lock()
			delete(c.peers, peer.id)
			c.lock.Unlock()

			logger.Infof("lost connection from replica %s (%s): %s", peer.id.Name, pi, err)
		}
	}

	return err
}

// stack interface
func (c *Backend) Broadcast(msg *pb.Message, peerType pb.PeerEndpoint_Type) error {
	panic("not implemented")
}

func (c *Backend) Unicast(msg *pb.Message, dest *pb.PeerID) error {
	c.lock.Lock()
	ch, ok := c.peers[*dest]
	c.lock.Unlock()

	if !ok {
		err := fmt.Errorf("peer not found: %v", dest)
		logger.Debug(err)
		return err
	}

	// XXX nonblocking interface
	ch <- msg

	return nil
}

//
func (c *Backend) GetNetworkInfo() (self *pb.PeerEndpoint, network []*pb.PeerEndpoint, err error) {
	panic("not implemented")
}

func (c *Backend) GetNetworkHandles() (self *pb.PeerID, network []*pb.PeerID, err error) {
	selfCopy := c.self.id
	self = &selfCopy
	for _, peer := range c.peerInfo {
		peer := peer
		network = append(network, &peer.id)
	}
	return
}
