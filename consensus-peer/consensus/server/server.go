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

package server

import (
	"sync"

	fabric_consensus "github.com/hyperledger/fabric/consensus"
	"github.com/hyperledger/fabric/consensus-peer/connection"
	"github.com/hyperledger/fabric/consensus-peer/consensus"
	"github.com/hyperledger/fabric/consensus/executor"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/op/go-logging"

	"golang.org/x/net/context"

	google_protobuf "google/protobuf"
)

var logger = logging.MustGetLogger("server")

type Server struct {
	fabric_consensus.Executor

	consensus fabric_consensus.Consenter
	queueSize int

	lock    sync.Mutex
	deliver map[chan *consensus.Block]struct{}

	executed *consensus.Block
}

type clientServer Server

func New(queueSize int, conn *connection.Manager) *Server {
	c := &Server{
		queueSize: queueSize,
		executed:  &consensus.Block{},
		deliver:   make(map[chan *consensus.Block]struct{}),
	}
	consensus.RegisterAtomicBroadcastServer(conn.Server, (*clientServer)(c))
	return c
}

func (c *Server) RegisterConsenter(consensus fabric_consensus.Consenter) {
	c.consensus = consensus
	c.Executor = executor.NewImpl(c.consensus, c, c)
	c.Executor.Start()
}

// gRPC atomic_broadcast interface
func (c *clientServer) Broadcast(ctx context.Context, msg *consensus.Message) (*google_protobuf.Empty, error) {
	// XXX check ctx tls credentials for permission to broadcast
	// XXX log broadcast
	c.consensus.RecvRequest(&pb.Transaction{Payload: msg.Data})
	return &google_protobuf.Empty{}, nil
}

func (c *clientServer) Deliver(_ *google_protobuf.Empty, srv consensus.AtomicBroadcast_DeliverServer) error {
	// XXX check src tls credentials for permission to subscribe
	// XXX log subscription
	ch := make(chan *consensus.Block, c.queueSize)
	c.lock.Lock()
	c.deliver[ch] = struct{}{}
	c.lock.Unlock()
	for msg := range ch {
		srv.Send(msg)
	}
	return nil
}

//
func (c *Server) BeginTxBatch(id interface{}) error {
	c.executed = &consensus.Block{}
	return nil
}

func (c *Server) ExecTxs(id interface{}, txs []*pb.Transaction) ([]byte, error) {
	msgs := make([]*consensus.Message, len(txs))
	for i, tx := range txs {
		msgs[i] = &consensus.Message{Data: tx.Payload}
	}
	c.executed.Messages = append(c.executed.Messages, msgs...)
	return nil, nil
}

func (c *Server) CommitTxBatch(id interface{}, metadata []byte) (*pb.Block, error) {
	b := c.executed
	c.executed = &consensus.Block{}

	c.lock.Lock()
	out := c.deliver
	c.lock.Unlock()

	for ch := range out {
		select {
		case ch <- b:
			break
		default:
			// ch is full, disconnect
			c.lock.Lock()
			delete(c.deliver, ch)
			c.lock.Unlock()
			close(ch)
		}
	}

	return nil, nil
}

func (c *Server) RollbackTxBatch(id interface{}) error {
	return nil
}

func (c *Server) PreviewCommitTxBatch(id interface{}, metadata []byte) ([]byte, error) {
	panic("not implemented")
}

func (c *Server) GetBlockchainInfo() *pb.BlockchainInfo {
	//	panic("not implemented")
	return &pb.BlockchainInfo{}
}

// Syncer
func (c *Server) Start() {
}

func (c *Server) Stop() {
	panic("not implemented")
}

func (c *Server) SyncToTarget(blockNumber uint64, blockHash []byte, peerIDs []*pb.PeerID) (error, bool) {
	panic("not implemented")
}

//
func (c *Server) GetBlockchainInfoBlob() []byte {
	// XXX assemble state for consensus service
	// XXX this probably should include last block hash, etc.
	//panic("not implemented")
	return []byte("some internal state")
}
