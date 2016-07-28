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

package peer

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/protos"
	"golang.org/x/net/context"
)

func mkErrResponse(err error) *protos.Response {
	return &protos.Response{
		Status: protos.Response_FAILURE,
		Msg:    []byte(fmt.Sprintf("Error: %s", err)),
	}
}

//ExecuteTransaction executes transactions decides to do execute in dev or prod mode
func (p *PeerImpl) ExecuteTransaction(transaction *protos.Transaction) (response *protos.Response) {
	p.ledgerWrapper.RLock()
	defer p.ledgerWrapper.RUnlock()

	chain := chaincode.GetChain(chaincode.DefaultChain)
	ctx := context.TODO()

	if transaction.Type == protos.Transaction_CHAINCODE_QUERY {
		result, _, err := chaincode.Execute(ctx, chain, transaction)
		if err != nil {
			return mkErrResponse(err)
		}
		return &protos.Response{
			Status: protos.Response_SUCCESS,
			Msg:    result,
		}
	}

	// else it is a write transaction

	// XXX actually "endorser"
	if !p.isValidator {
		peerAddresses := p.discHelper.GetRandomNodes(1)
		response = p.SendTransactionsToPeer(peerAddresses[0], transaction)
	}

	id := struct{}{}
	ledger := p.ledgerWrapper.ledger
	ledger.BeginTxBatch(id)
	_, events, err := chaincode.Execute(ctx, chain, transaction)
	_ = events // when do those get emitted?
	if err != nil {
		ledger.RollbackTxBatch(id)
		return mkErrResponse(err)
	}
	changeset := ledger.GetTxChangeset()
	ledger.RollbackTxBatch(id)

	// XXX do the endorsement part

	err = p.engine.NewChangeset(changeset)
	if err != nil {
		return mkErrResponse(err)
	}

	return &protos.Response{
		Status: protos.Response_SUCCESS,
		Msg:    []byte(transaction.Uuid),
	}
}

func (p *PeerImpl) CreateValidBlock(block *protos.Block) *protos.Block {
	// XXX check changesets in block
	return block
}

func (p *PeerImpl) ProposeBlock(block *protos.Block) {
	// XXX check validity of block
	// XXX gossip about block
	// XXX optionally, sign block and gossip signature
	// XXX optionally, check for threshold of signatures
	p.ledgerWrapper.RLock()
	defer p.ledgerWrapper.RUnlock()
	p.ledgerWrapper.ledger.AppendBlock(block)
}
