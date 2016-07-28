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

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/consensus-peer/consensus"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/op/go-logging"
	"github.com/spf13/viper"

	"golang.org/x/net/context"
)

var logger = logging.MustGetLogger("engine")

type LedgerPeer interface {
	CreateValidBlock(block *pb.Block) *pb.Block
	ProposeBlock(block *pb.Block)
}

// Engine implements a struct to hold consensus.Consenter, PeerEndpoint and MessageFan
type Engine struct {
	peer      LedgerPeer
	consensus consensus.AtomicBroadcastClient
}

// ProcessTransactionMsg processes a Message in context of a Transaction
func (eng *Engine) NewChangeset(cs *pb.Changeset) error {
	csRaw, err := proto.Marshal(cs)
	if err != nil {
		return err
	}

	// TODO, This will block until the consenter gets
	// around to handling the message, but it also
	// provides some natural feedback to the REST API to
	// determine how long it takes to queue messages
	ctx := context.TODO()
	_, err = eng.consensus.Broadcast(ctx, &consensus.Message{Data: csRaw})
	return err
}

func (e *Engine) deliverHandler(ctx context.Context) {
RECONNECT:
	for {
		select {
		case <-ctx.Done():
			break RECONNECT
		default:
		}

		client, err := e.consensus.Deliver(ctx, &google_protobuf.Empty{})
		if err != nil {
			// XXX can this happen?
			panic(err)
		}

		for {
			block, err := client.Recv()
			if err != nil {
				logger.Warningf("receive from consensus peer: %s", err)
				break
			}

			err = e.processBlock(ctx, block)
			if err != nil {
				logger.Errorf("cannot process new transaction block: %s", err)
				break
			}
		}
	}
}

func (e *Engine) processBlock(ctx context.Context, block *consensus.Block) error {
	msgs := block.GetMessages()
	txs := make([]*pb.Changeset, len(msgs))
	for i, m := range msgs {
		tx := &pb.Changeset{}
		err := proto.Unmarshal(m.Data, tx)
		if err != nil {
			// XXX drop consenter, redial, state sync?
			panic(err)
		}
		txs[i] = tx
	}

	b := &pb.Block{
		Changes:           txs,
		PreviousBlockHash: []byte("XXX"),
	}

	b = e.peer.CreateValidBlock(b)
	e.peer.ProposeBlock(b)

	return nil
}

// New creates a new Engine
func New(p LedgerPeer) (_ *Engine, err error) {
	e := &Engine{peer: p}

	addr := viper.GetString("peer.validator.consenter.address")
	cert := viper.GetString("peer.validator.consenter.cert.file")

	if addr == "local-development-loopback-consensus" {
		e.consensus = newLoopbackConsensus()
	} else {
		e.consensus, err = consensus.DialPem(addr, cert)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.TODO()
	go e.deliverHandler(ctx)

	return e, nil
}
