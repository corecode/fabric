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

package client

import (
	"sync"

	"github.com/hyperledger/fabric/consensus"
	"github.com/hyperledger/fabric/consensus-peer/connection"
	pb "github.com/hyperledger/fabric/protos"

	"golang.org/x/net/context"

	google_protobuf "google/protobuf"
)

type Client struct {
	consensus consensus.Consenter
	queueSize int

	lock    sync.Mutex
	deliver map[chan *Block]struct{}

	executed *Block
}

type clientConn Client

func New(queueSize int, conn *connection.Manager) *Client {
	c := &Client{
		queueSize: queueSize,
		executed:  &Block{},
	}
	RegisterAtomicBroadcastServer(conn.Server, (*clientConn)(c))
	return c
}

func (c *Client) RegisterConsensus(consensus consensus.Consenter) {
	c.consensus = consensus
}

// gRPC atomic_broadcast interface
func (c *clientConn) Broadcast(ctx context.Context, msg *Message) (*google_protobuf.Empty, error) {
	// XXX check ctx tls credentials for permission to broadcast
	c.consensus.RecvRequest(&pb.Transaction{Payload: msg.Data})
	return nil, nil
}

func (c *clientConn) Deliver(_ *google_protobuf.Empty, srv AtomicBroadcast_DeliverServer) error {
	// XXX check src tls credentials for permission to subscribe
	ch := make(chan *Block, c.queueSize)
	c.lock.Lock()
	c.deliver[ch] = struct{}{}
	c.lock.Unlock()
	for msg := range ch {
		srv.Send(msg)
	}
	return nil
}

// consensus.Executor interface
func (c *Client) Start() {
	panic("not implemented")
}

func (c *Client) Halt() {
	panic("not implemented")
}

func (c *Client) Execute(tag interface{}, txs []*pb.Transaction) {
	msgs := make([]*Message, len(txs))
	for i, tx := range txs {
		msgs[i] = &Message{Data: tx.Payload}
	}
	c.executed.Messages = append(c.executed.Messages, msgs...)
}

func (c *Client) Commit(tag interface{}, metadata []byte) {
	b := c.executed
	c.executed = &Block{}

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
}

func (c *Client) Rollback(tag interface{}) {
	c.executed = &Block{}
}

func (c *Client) UpdateState(tag interface{}, target *pb.BlockchainInfo, peers []*pb.PeerID) {
	panic("not implemented")
}

//
func (c *Client) BeginTxBatch(id interface{}) error {
	panic("not implemented")
}

func (c *Client) ExecTxs(id interface{}, txs []*pb.Transaction) ([]byte, error) {
	panic("not implemented")
}

func (c *Client) CommitTxBatch(id interface{}, metadata []byte) (*pb.Block, error) {
	panic("not implemented")
}

func (c *Client) RollbackTxBatch(id interface{}) error {
	panic("not implemented")
}

func (c *Client) PreviewCommitTxBatch(id interface{}, metadata []byte) ([]byte, error) {
	panic("not implemented")
}
