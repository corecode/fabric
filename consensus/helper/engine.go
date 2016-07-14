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
	"fmt"
	"google/protobuf"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/consensus-peer/consensus"
	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/core/ledger"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/op/go-logging"
	"github.com/spf13/viper"

	"golang.org/x/net/context"
)

var logger = logging.MustGetLogger("engine")

// Engine implements a struct to hold consensus.Consenter, PeerEndpoint and MessageFan
type Engine struct {
	consensus consensus.AtomicBroadcastClient
}

func mkResponse(msg []byte, err error) *pb.Response {
	r := &pb.Response{Status: pb.Response_SUCCESS}
	if err != nil {
		msg = []byte(fmt.Sprintf("Error: %s", err))
		r.Status = pb.Response_FAILURE
	}
	r.Msg = msg
	return r
}

// ProcessTransactionMsg processes a Message in context of a Transaction
func (eng *Engine) ProcessTransactionMsg(tx *pb.Transaction) (response *pb.Response) {
	//TODO: Do we always verify security, or can we supply a flag on the invoke ot this functions so to bypass check for locally generated transactions?
	if tx.Type == pb.Transaction_CHAINCODE_QUERY {
		// The secHelper is set during creat ChaincodeSupport,
		// so we don't need this step cxt :=
		// context.WithValue(context.Background(), "security",
		// secHelper)
		cxt := context.TODO()
		//query will ignore events as these are not stored on
		//ledger (and query can report "event" data
		//synchronously anyway)
		result, _, err := chaincode.Execute(cxt, chaincode.GetChain(chaincode.DefaultChain), tx)
		response = mkResponse(result, err)
	} else {
		//TODO: Do we need to verify security, or can we
		//supply a flag on the invoke ot this functions If we
		//fail to marshal or verify the tx, don't send it to
		//consensus plugin

		txRaw, err := proto.Marshal(tx)
		if err != nil {
			return mkResponse(nil, err)
		}

		// TODO, This will block until the consenter gets
		// around to handling the message, but it also
		// provides some natural feedback to the REST API to
		// determine how long it takes to queue messages
		ctx := context.TODO()
		_, err = eng.consensus.Broadcast(ctx, &consensus.Message{Data: txRaw})
		response = mkResponse([]byte(tx.Uuid), err)
	}
	return response
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
	ledger, err := ledger.GetLedger()
	if err != nil {
		// this is an unrecoverable error
		panic(err)
	}

	id := block

	err = ledger.BeginTxBatch(id)
	if err != nil {
		// sequencing error?  we can't recover really
		panic(err)
	}

	msgs := block.GetMessages()
	txs := make([]*pb.Transaction, len(msgs))
	for i, m := range msgs {
		tx := &pb.Transaction{}
		err := proto.Unmarshal(m.Data, tx)
		if err != nil {
			// XXX drop consenter, redial, state sync?
			panic(err)
		}
		txs[i] = tx
	}

	successTXs, _, ccevents, txerrs, err := chaincode.ExecuteTransactions(ctx, chaincode.DefaultChain, txs)

	// XXX why doesn't ExecuteTransactions return TransactionResult?
	txresults := make([]*pb.TransactionResult, len(txerrs))
	for i, e := range txerrs {
		res := &pb.TransactionResult{
			Uuid:           txs[i].Uuid,
			ChaincodeEvent: ccevents[i],
		}
		if e != nil {
			res.Error = e.Error()
			res.ErrorCode = 1
		}
		txresults[i] = res
	}

	err = ledger.CommitTxBatch(id, successTXs, txresults, nil)
	if err != nil {
		// sequencing error?  we can't recover really
		panic(err)
	}
	return nil
}

// New creates a new Engine
func New() (_ *Engine, err error) {
	e := &Engine{}

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
