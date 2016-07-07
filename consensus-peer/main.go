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
	"flag"
	"fmt"
	"os"

	"github.com/hyperledger/fabric/consensus-peer/backend"
	"github.com/hyperledger/fabric/consensus-peer/connection"
	"github.com/hyperledger/fabric/consensus-peer/consensus/server"
	"github.com/hyperledger/fabric/consensus-peer/persist"
	"github.com/hyperledger/fabric/consensus/pbft"

	pb "github.com/hyperledger/fabric/protos"
)

type consensusStack struct {
	*persist.Persist
	*server.Server
	*backend.Backend
}

//
func (c *consensusStack) Sign(msg []byte) ([]byte, error) {
	//panic("not implemented")
	return nil, nil
}

func (c *consensusStack) Verify(peerID *pb.PeerID, signature []byte, message []byte) error {
	//panic("not implemented")
	return nil
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

func (c *consensusStack) GetBlockchainInfoBlob() []byte {
	panic("not implemented")
}

func (c *consensusStack) GetBlockHeadMetadata() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

type config struct {
	listenAddr string
	certFile   string
	keyFile    string
	dataDir    string
}

//
func main() {
	var c config

	flag.StringVar(&c.listenAddr, "addr", ":6100", "`addr`ess/port of service")
	flag.StringVar(&c.certFile, "cert", "", "certificate `file`")
	flag.StringVar(&c.keyFile, "key", "", "key `file`")
	flag.StringVar(&c.dataDir, "data-dir", "", "data `dir`ectory")

	flag.Parse()

	if c.dataDir == "" {
		fmt.Fprintln(os.Stderr, "need data directory")
		os.Exit(1)
	}

	conn, err := connection.New(c.listenAddr, c.certFile, c.keyFile)
	if err != nil {
		panic(err)
	}
	s := &consensusStack{
		Persist: persist.New(c.dataDir),
		Server:  server.New(100, conn),
	}
	s.Backend, err = backend.New(s.Persist, conn)
	if err != nil {
		panic(err)
	}

	pbft := pbft.New(s)
	s.Server.RegisterConsenter(pbft)
	s.Backend.RegisterConsenter(pbft)

	// block forever
	select {}
}
