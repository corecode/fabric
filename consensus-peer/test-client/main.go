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
	"google/protobuf"
	"time"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	"github.com/hyperledger/fabric/consensus-peer/consensus"
)

var (
	certFile  = flag.String("cert", "", "certificate `file`")
	addr      = flag.String("addr", "", "consensus `peer` address")
	listen    = flag.Bool("listen", false, "whether to listen to delivery events")
	broadcast = flag.String("broadcast", "", "string to broadcast")
)

func main() {
	flag.Parse()

	c, err := consensus.DialPem(*addr, *certFile, grpc.WithBlock(), grpc.WithTimeout(time.Second))
	if err != nil {
		panic(err)
	}
	fmt.Println("connection established")

	if *listen {
		client, err := c.Deliver(context.TODO(), &google_protobuf.Empty{})
		if err != nil {
			panic(err)
		}

		go func() {
			for {
				block, err := client.Recv()
				if err != nil {
					panic(err)
				}
				fmt.Printf("received block:\n")
				for i, msg := range block.GetMessages() {
					fmt.Printf("%d: %s\n", i, string(msg.Data))
				}
			}
		}()
	}

	if *broadcast != "" {
		c.Broadcast(context.TODO(), &consensus.Message{[]byte(*broadcast)})
	}

	if *listen {
		select {}
	}
}
