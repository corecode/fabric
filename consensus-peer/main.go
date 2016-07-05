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

package main

import (
	"fmt"

	"github.com/hyperledger/fabric/consensus-peer/connection"
	"github.com/hyperledger/fabric/consensus-peer/consensus"
	"github.com/hyperledger/fabric/consensus-peer/persist"
	"github.com/hyperledger/fabric/consensus/pbft"

	pb "github.com/hyperledger/fabric/protos"
)

type consensusStack struct {
	*persist.Persist
	*connection.Conn
	*consensus.Consensus
}

//
func (c *consensusStack) Sign(msg []byte) ([]byte, error) {
	panic("not implemented")
}

func (c *consensusStack) Verify(peerID *pb.PeerID, signature []byte, message []byte) error {
	panic("not implemented")
}

//
func (c *consensusStack) InvalidateState() {
	panic("not implemented")
}

func (c *consensusStack) ValidateState() {
	panic("not implemented")
}

//
func (c *consensusStack) GetBlock(id uint64) (block *pb.Block, err error) {
	panic("not implemented")
}

func (c *consensusStack) GetBlockchainSize() uint64 {
	panic("not implemented")
}

func (c *consensusStack) GetBlockchainInfo() *pb.BlockchainInfo {
	panic("not implemented")
}

func (c *consensusStack) GetBlockchainInfoBlob() []byte {
	panic("not implemented")
}

func (c *consensusStack) GetBlockHeadMetadata() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

//
func main() {
	s := &consensusStack{
		Persist:   persist.New("data"),
		Conn:      connection.New(100),
		Consensus: consensus.New(consensus.PeerInfo{}, nil),
	}
	s.RegisterConsenter(pbft.New(s))
	for {
	}
}
