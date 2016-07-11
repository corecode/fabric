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

package consensus

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"github.com/hyperledger/fabric/consensus-peer/connection"
	"google.golang.org/grpc"
)

func Dial(addr string, cert []byte, opts ...grpc.DialOption) (AtomicBroadcastClient, error) {
	peer, err := connection.NewPeerInfo(addr, cert)
	if err != nil {
		return nil, err
	}

	conn, err := connection.DialPeer(peer, opts...)
	if err != nil {
		return nil, err
	}

	client := NewAtomicBroadcastClient(conn)

	return client, nil
}

func DialPem(addr string, certFile string, opts ...grpc.DialOption) (AtomicBroadcastClient, error) {
	certBytes, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	var b *pem.Block
	for {
		b, certBytes = pem.Decode(certBytes)
		if b == nil {
			break
		}
		if b.Type == "CERTIFICATE" {
			break
		}
	}

	if b == nil {
		return nil, fmt.Errorf("no certificate found")
	}

	return Dial(addr, b.Bytes, opts...)
}
