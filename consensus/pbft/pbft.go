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

package pbft

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric/consensus"
	pb "github.com/hyperledger/fabric/protos"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/viper"
)

const configPrefix = "CORE_PBFT"

// New creates a new Obc* instance that provides the Consenter interface.
// Internally, it uses an opaque pbft-core instance.
func New(config *BatchConfig, stack consensus.Stack) *obcBatch {
	handle, _, _ := stack.GetNetworkHandles()
	id, _ := getValidatorID(handle)

	return newObcBatch(id, config, stack)
}

func loadConfig() BatchConfig {
	config := viper.New()

	// for environment variables
	config.SetEnvPrefix(configPrefix)
	config.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	config.SetEnvKeyReplacer(replacer)

	config.SetConfigName("config")
	config.AddConfigPath("./")
	config.AddConfigPath("../consensus/pbft/")
	config.AddConfigPath("../../consensus/pbft")
	// Path to look for the config file in based on GOPATH
	gopath := os.Getenv("GOPATH")
	for _, p := range filepath.SplitList(gopath) {
		pbftpath := filepath.Join(p, "src/github.com/hyperledger/fabric/consensus/pbft")
		config.AddConfigPath(pbftpath)
	}

	err := config.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Error reading %s plugin config: %s", configPrefix, err))
	}

	bconfig := BatchConfig{
		PbftConfig: &PbftConfig{
			N:                       uint64(config.GetInt("general.N")),
			F:                       uint64(config.GetInt("general.f")),
			K:                       uint64(config.GetInt("general.K")),
			LMultiplier:             uint64(config.GetInt("general.logmultiplier")),
			ViewchangePeriod:        uint64(config.GetInt("general.viewchangeperiod")),
			RequestTimeout:          uint64(config.GetDuration("general.timeout.request")),
			ViewchangeResendTimeout: uint64(config.GetDuration("general.timeout.resendviewchange")),
			ViewchangeTimeout:       uint64(config.GetDuration("general.timeout.viewchange")),
			NullRequestTimeout:      uint64(config.GetDuration("general.timeout.nullrequest")),
		},
		BatchSize:    uint64(config.GetInt("general.batchsize")),
		BatchTimeout: uint64(config.GetDuration("general.timeout.batch")),
		Outstanding:  uint64(config.GetInt("general.outstanding")),
	}
	return bconfig
}

// Returns the uint64 ID corresponding to a peer handle
func getValidatorID(handle *pb.PeerID) (id uint64, err error) {
	// as requested here: https://github.com/hyperledger/fabric/issues/462#issuecomment-170785410
	if startsWith := strings.HasPrefix(handle.Name, "vp"); startsWith {
		id, err = strconv.ParseUint(handle.Name[2:], 10, 64)
		if err != nil {
			return id, fmt.Errorf("Error extracting ID from \"%s\" handle: %v", handle.Name, err)
		}
		return
	}

	err = fmt.Errorf(`For MVP, set the VP's peer.id to vpX,
		where X is a unique integer between 0 and N-1
		(N being the maximum number of VPs in the network`)
	return
}

// Returns the peer handle that corresponds to a validator ID (uint64 assigned to it for PBFT)
func getValidatorHandle(id uint64) (handle *pb.PeerID, err error) {
	// as requested here: https://github.com/hyperledger/fabric/issues/462#issuecomment-170785410
	name := "vp" + strconv.FormatUint(id, 10)
	return &pb.PeerID{Name: name}, nil
}

// Returns the peer handles corresponding to a list of replica ids
func getValidatorHandles(ids []uint64) (handles []*pb.PeerID) {
	handles = make([]*pb.PeerID, len(ids))
	for i, id := range ids {
		handles[i], _ = getValidatorHandle(id)
	}
	return
}

type obcGeneric struct {
	stack consensus.Stack
	pbft  *pbftCore
}

func (op *obcGeneric) skipTo(seqNo uint64, id []byte, replicas []uint64) {
	info := &pb.BlockchainInfo{}
	err := proto.Unmarshal(id, info)
	if err != nil {
		logger.Error(fmt.Sprintf("Error unmarshaling: %s", err))
		return
	}
	op.stack.UpdateState(&checkpointMessage{seqNo, id}, info, getValidatorHandles(replicas))
}

func (op *obcGeneric) invalidateState() {
	op.stack.InvalidateState()
}

func (op *obcGeneric) validateState() {
	op.stack.ValidateState()
}

func (op *obcGeneric) getState() []byte {
	return op.stack.GetBlockchainInfoBlob()
}

func (op *obcGeneric) getLastSeqNo() (uint64, error) {
	raw, err := op.stack.GetBlockHeadMetadata()
	if err != nil {
		return 0, err
	}
	meta := &Metadata{}
	proto.Unmarshal(raw, meta)
	return meta.SeqNo, nil
}
