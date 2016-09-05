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
	"time"

	"github.com/golang/protobuf/proto"
)

func (s *SBFT) sendPreprepare(batch []*Request) {
	seq := s.nextSeq()

	set := &DigestSet{Digest: make([][]byte, len(batch))}

	for i, req := range batch {
		set.Digest[i] = req.Payload
	}

	rawSet, err := proto.Marshal(set)
	if err != nil {
		panic(err)
	}

	m := &Preprepare{
		Seq: &seq,
		Set: rawSet,
	}

	s.broadcast(&Msg{&Msg_Preprepare{m}})
}

func (s *SBFT) handlePreprepare(pp *Preprepare, src uint64) {
	if src != s.primaryId() {
		log.Infof("preprepare from non-primary %d", src)
		return
	}
	nextSeq := s.nextSeq()
	if *pp.Seq != nextSeq {
		log.Infof("preprepare does not match expected %v, got %v", nextSeq, *pp.Seq)
		return
	}
	h := s.hash(pp.Set)

	payload := &DigestSet{}
	err := proto.Unmarshal(pp.Set, payload)
	if err != nil {
		log.Infof("preprepare digest set malformed in preprepare %v from %s, %s", s.seq, src, h)
		return
	}

	s.acceptPreprepare(Subject{Seq: &nextSeq, Digest: h}, payload, pp)
}

func (s *SBFT) acceptPreprepare(sub Subject, payload *DigestSet, pp *Preprepare) {
	s.cur = reqInfo{
		subject:    sub,
		timeout:    s.sys.Timer(time.Duration(s.config.RequestTimeoutNsec)*time.Nanosecond, s.requestTimeout),
		payload:    payload,
		preprep:    pp,
		prep:       make(map[uint64]*Subject),
		commit:     make(map[uint64]*Subject),
		checkpoint: make(map[uint64]*Checkpoint),
	}

	log.Infof("accepting preprepare for %v, %x", s.seq, sub.Digest)
	s.cancelViewChangeTimer()
	if !s.isPrimary() {
		s.sendPrepare()
	}

	s.maybeSendCommit()
}

////////////////////////////////////////////////

func (s *SBFT) requestTimeout() {
	log.Infof("request timed out: %s", s.cur.subject.Seq)
	s.sendViewChange()
}
