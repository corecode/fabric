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
	"google/protobuf"
	"io"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/hyperledger/fabric/consensus"
	pb "github.com/hyperledger/fabric/protos"

	"google.golang.org/grpc"
)

type Consensus struct {
	consensus consensus.Consenter

	lock  sync.Mutex
	peers map[pb.PeerID]chan<- *pb.Message

	self     PeerInfo
	peerInfo []PeerInfo
}

type consensusConn Consensus

type PeerInfo struct {
	pb.PeerID
}

func New(self PeerInfo, peerInfo []PeerInfo) *Consensus {
	c := &Consensus{
		peers:    make(map[pb.PeerID]chan<- *pb.Message),
		self:     self,
		peerInfo: peerInfo,
	}

	for _, peer := range c.peerInfo {
		if peer == c.self {
			continue
		}
		go c.connectWorker(peer)
	}
	return c
}

func (c *Consensus) RegisterConsenter(consensus consensus.Consenter) {
	c.consensus = consensus
}

func (c *Consensus) connectWorker(peer PeerInfo) {
	delay := time.After(0)
	for {
		// pace reconnect attempts
		<-delay

		// set up for next
		delay = time.After(1 * time.Second)

		// XXX get address
		addr := peer.Name

		// XXX pass context via NewConsensus
		ctx := context.TODO()

		// XXX tls dialopts
		conn, err := grpc.Dial(addr)
		if err != nil {
			panic(err)
			continue
		}

		client := NewConsensusClient(conn)
		consensus, err := client.Consensus(ctx, nil)
		if err != nil {
			panic(err)
			continue
		}

		for {
			msg, err := consensus.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				panic(err)
			}
			c.consensus.RecvMsg(msg, &peer.PeerID)
		}
		conn.Close()
	}
}

// gRPC interface
func (c *consensusConn) Consensus(_ *google_protobuf.Empty, srv Consensus_ConsensusServer) error {
	// XXX check tls cert
	// XXX map to peerid
	peer := c.peerInfo[0]

	ch := make(chan *pb.Message)
	c.lock.Lock()
	if oldch, ok := c.peers[peer.PeerID]; ok {
		close(oldch)
	}
	c.peers[peer.PeerID] = ch
	c.lock.Unlock()

	var err error
	for msg := range ch {
		err = srv.Send(msg)
		if err != nil {
			break
		}
	}

	c.lock.Lock()
	delete(c.peers, peer.PeerID)
	c.lock.Lock()

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
	selfCopy := c.self.PeerID
	self = &selfCopy
	for _, peer := range c.peerInfo {
		peer := peer
		network = append(network, &peer.PeerID)
	}
	return
}
