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

package helper

import (
	"google/protobuf"

	"github.com/hyperledger/fabric/consensus-peer/consensus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type loopbackConsensus struct {
	ch chan *consensus.Message
}

func newLoopbackConsensus() *loopbackConsensus {
	return &loopbackConsensus{make(chan *consensus.Message)}
}

func (c *loopbackConsensus) Broadcast(ctx context.Context, msg *consensus.Message, opts ...grpc.CallOption) (*google_protobuf.Empty, error) {
	select {
	case c.ch <- msg:
		return &google_protobuf.Empty{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *loopbackConsensus) BroadcastStream(ctx context.Context, opts ...grpc.CallOption) (consensus.AtomicBroadcast_BroadcastStreamClient, error) {
	return &loopbackConsensusBroadcast{c, make(chan struct{}, 10)}, nil
}

type loopbackConsensusBroadcast struct {
	*loopbackConsensus
	bcastC chan struct{}
}

func (c *loopbackConsensusBroadcast) Send(msg *consensus.Message) error {
	c.ch <- msg
	c.bcastC <- struct{}{}
	return nil
}

func (c *loopbackConsensusBroadcast) Recv() (*consensus.Reply, error) {
	<-c.bcastC
	return &consensus.Reply{}, nil
}

func (c *loopbackConsensus) Deliver(ctx context.Context, in *google_protobuf.Empty, opts ...grpc.CallOption) (consensus.AtomicBroadcast_DeliverClient, error) {
	return &loopbackConsensusDeliver{c}, nil
}

type loopbackConsensusDeliver struct {
	*loopbackConsensus
}

func (c *loopbackConsensusDeliver) Recv() (*consensus.Block, error) {
	msg := <-c.ch
	return &consensus.Block{Messages: []*consensus.Message{msg}}, nil
}

func (c *loopbackConsensus) CloseSend() error             { panic("not implemented") }
func (c *loopbackConsensus) Header() (metadata.MD, error) { panic("not implemented") }
func (c *loopbackConsensus) Trailer() metadata.MD         { panic("not implemented") }
func (c *loopbackConsensus) Context() context.Context     { panic("not implemented") }
func (c *loopbackConsensus) SendMsg(m interface{}) error  { panic("not implemented") }
func (c *loopbackConsensus) RecvMsg(m interface{}) error  { panic("not implemented") }
