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
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/hyperledger/fabric/consensus-peer/backend"
	"github.com/hyperledger/fabric/consensus-peer/connection"
	"github.com/hyperledger/fabric/consensus-peer/consensus/server"
	"github.com/hyperledger/fabric/consensus-peer/persist"
	"github.com/hyperledger/fabric/consensus/pbft"
	"github.com/hyperledger/fabric/core/telemetry"
	"github.com/op/go-logging"

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

func (c *consensusStack) GetBlockHeadMetadata() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

type flags struct {
	listenAddr    string
	telemetryAddr string
	certFile      string
	keyFile       string
	dataDir       string
	verbose       string
	init          string
}

//
func main() {
	var c flags

	flag.StringVar(&c.init, "init", "", "initialized instance from pbft config `file`")

	flag.StringVar(&c.listenAddr, "addr", ":6100", "`addr`ess/port of service")
	flag.StringVar(&c.telemetryAddr, "telemetry", ":7100", "`addr`ess of telemetry/profiler")
	flag.StringVar(&c.certFile, "cert", "", "certificate `file`")
	flag.StringVar(&c.keyFile, "key", "", "key `file`")
	flag.StringVar(&c.dataDir, "data-dir", "", "data `dir`ectory")
	flag.StringVar(&c.verbose, "verbose", "info", "set verbosity `level` (critical, error, warning, notice, info, debug)")

	flag.Parse()

	level, err := logging.LogLevel(c.verbose)
	if err != nil {
		panic(err)
	}
	logging.SetLevel(level, "")

	if c.init != "" {
		err = initInstance(c)
		if err != nil {
			panic(err)
		}
		return
	}

	serve(c)
}

func initInstance(c flags) error {
	config, err := ReadJsonConfig(c.init)
	if err != nil {
		return err
	}

	err = os.Mkdir(c.dataDir, 0755)
	if err != nil {
		return err
	}

	p := persist.New(c.dataDir)
	err = SaveConfig(p, config)
	if err != nil {
		return err
	}

	fmt.Println("initialized new peer")
	return nil
}

func serve(c flags) {
	if c.dataDir == "" {
		fmt.Fprintln(os.Stderr, "need data directory")
		os.Exit(1)
	}

	persist := persist.New(c.dataDir)
	config, err := RestoreConfig(persist)
	if err != nil {
		panic(err)
	}

	if c.telemetryAddr != "" {
		go func() {
			err := http.ListenAndServe(c.telemetryAddr, nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}()
		telemetry.Init("/telemetry", nil)
	}

	conn, err := connection.New(c.listenAddr, c.certFile, c.keyFile)
	if err != nil {
		panic(err)
	}
	s := &consensusStack{
		Persist: persist,
		Server:  server.New(100, conn),
	}
	s.Backend, err = backend.New(config.Peers, conn)
	if err != nil {
		panic(err)
	}

	pbft := pbft.New(config.Consensus, s)
	s.Backend.RegisterConsenter(pbft)
	s.Server.RegisterConsenter(pbft)

	// block forever
	select {}
}
