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

package obcpbft

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/util"
	pb "github.com/hyperledger/fabric/protos"
)

type reqQueue struct {
	l           sync.Mutex
	Ready       chan int
	outstanding map[string]int
}

func newReqQueue(count int) *reqQueue {
	c := &reqQueue{
		Ready:       make(chan int, count),
		outstanding: make(map[string]int),
	}
	for i := 0; i < count; i++ {
		c.Ready <- i
	}
	return c
}

func (c *reqQueue) txID(tx *pb.Transaction) string {
	raw, _ := proto.Marshal(tx)
	return base64.StdEncoding.EncodeToString(util.ComputeCryptoHash(raw))
}

func (c *reqQueue) Register(tx *pb.Transaction) (int, error) {
	id := c.txID(tx)
	i := <-c.Ready
	c.l.Lock()
	defer c.l.Unlock()
	if _, ok := c.outstanding[id]; ok {
		return -1, fmt.Errorf("duplicate tx %s", id)
	}
	c.outstanding[id] = i
	return i, nil
}

func (c *reqQueue) Done(tx *pb.Transaction) {
	id := c.txID(tx)
	c.l.Lock()
	defer c.l.Unlock()
	i, ok := c.outstanding[id]
	if !ok {
		// transaction originating from other replica
		return
	}
	delete(c.outstanding, id)
	c.Ready <- i
}

func (c *reqQueue) Clear() {
	c.l.Lock()
	defer c.l.Unlock()
	for id, i := range c.outstanding {
		delete(c.outstanding, id)
		c.Ready <- i
	}
}
