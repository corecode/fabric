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

package consensus

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/transport"

	"golang.org/x/net/context"

	"github.com/hyperledger/fabric/consensus"
	"github.com/hyperledger/fabric/consensus-peer/connection"
	"github.com/hyperledger/fabric/consensus-peer/persist"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("consensus")

type Consensus struct {
	consensus consensus.Consenter
	persist   *persist.Persist
	conn      *connection.Manager

	lock  sync.Mutex
	peers map[pb.PeerID]chan<- *pb.Message

	self     *PeerInfo
	peerInfo map[string]*PeerInfo
}

type consensusConn Consensus

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

func New(persist *persist.Persist, conn *connection.Manager) (*Consensus, error) {
	c := &Consensus{
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
		pi.id.Name = strconv.Itoa(i)
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

func (c *Consensus) RegisterConsenter(consensus consensus.Consenter) {
	c.consensus = consensus
}

func (c *Consensus) connectWorker(peer *PeerInfo) {
	delay := time.After(0)
	for {
		// pace reconnect attempts
		<-delay

		// set up for next
		delay = time.After(1 * time.Second)

		conn, err := c.conn.DialPeer(peer.info)
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
		logger.Infof("connection to replica %s (%s) established", peer.id.Name, peer.info)

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
		conn.Close()
	}
}

// gRPC interface
func (c *consensusConn) Consensus(_ *Handshake, srv Consensus_ConsensusServer) error {
	pi := c.conn.GetPeer(srv)
	peer, ok := c.peerInfo[pi.Fingerprint()]

	if !ok {
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
func (c *Consensus) Broadcast(msg *pb.Message, peerType pb.PeerEndpoint_Type) error {
	panic("not implemented")
}

func (c *Consensus) Unicast(msg *pb.Message, dest *pb.PeerID) error {
	c.lock.Lock()
	ch, ok := c.peers[*dest]
	c.lock.Unlock()

	if !ok {
		return fmt.Errorf("peer not found: %v", dest)
	}

	// XXX nonblocking interface
	ch <- msg

	return nil
}

//
func (c *Consensus) GetNetworkInfo() (self *pb.PeerEndpoint, network []*pb.PeerEndpoint, err error) {
	panic("not implemented")
}

func (c *Consensus) GetNetworkHandles() (self *pb.PeerID, network []*pb.PeerID, err error) {
	selfCopy := c.self.id
	self = &selfCopy
	for _, peer := range c.peerInfo {
		peer := peer
		network = append(network, &peer.id)
	}
	return
}
