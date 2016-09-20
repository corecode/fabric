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

package simplebft

import (
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
)

func (s *SBFT) sendCheckpoint() {
	sig := s.sys.Sign(s.cur.subject.Digest)
	c := &Checkpoint{
		Seq:       s.cur.subject.Seq.Seq,
		Digest:    s.cur.subject.Digest,
		Signature: sig,
	}
	s.broadcast(&Msg{&Msg_Checkpoint{c}})
}

func (s *SBFT) handleCheckpoint(c *Checkpoint, src uint64) {
	if s.cur.checkpointDone {
		return
	}

	if c.Seq < s.cur.subject.Seq.Seq {
		// old message
		return
	}

	err := s.checkBytesSig(c.Digest, src, c.Signature)
	if err != nil {
		log.Infof("checkpoint signature invalid for %d from %d", c.Seq, src)
		return
	}

	// TODO should we always accept checkpoints?
	if c.Seq != s.cur.subject.Seq.Seq {
		log.Infof("checkpoint does not match expected subject %v, got %v", &s.cur.subject, c)
		return
	}
	if _, ok := s.cur.checkpoint[src]; ok {
		log.Infof("duplicate checkpoint for %d from %d", c.Seq, src)
	}
	s.cur.checkpoint[src] = c

	max := "_"
	sums := make(map[string][]uint64)
	for csrc, c := range s.cur.checkpoint {
		sum := fmt.Sprintf("%x", c.Digest)
		sums[sum] = append(sums[sum], csrc)

		if len(sums[sum]) >= s.oneCorrectQuorum() {
			max = sum
		}
	}

	replicas, ok := sums[max]
	if !ok || len(sums[max]) != s.oneCorrectQuorum() {
		return
	}

	// got a stable checkpoint

	cpset := &CheckpointSet{make(map[uint64]*Checkpoint)}
	var sigs [][]byte
	for _, r := range replicas {
		cp := s.cur.checkpoint[r]
		cpset.CheckpointSet[r] = cp
		sigs = append(sigs, cp.Signature)
	}
	s.sys.Persist("checkpoint", cpset)
	s.cur.checkpointDone = true

	c = s.cur.checkpoint[replicas[0]]

	if !reflect.DeepEqual(c.Digest, s.cur.subject.Digest) {
		log.Fatalf("weak checkpoint %x does not match our state %x",
			c.Digest, s.cur.subject.Digest)
		// NOT REACHED
	}

	// ignore null requests
	if s.cur.payload.Digest != nil {
		bh := &BatchHeader{}
		err = proto.Unmarshal(s.cur.preprep.BatchHeader, bh)
		if err != nil {
			panic(err)
		}
		s.sys.Deliver(bh, s.cur.payload.Digest, sigs)
	}

	s.maybeSendNextBatch()
	s.processBacklog()
}
