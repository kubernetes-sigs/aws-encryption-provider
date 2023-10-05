/*
Copyright 2020 The Kubernetes Authors.
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
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	pb "k8s.io/kms/apis/v2"
	"sigs.k8s.io/aws-encryption-provider/pkg/connection"
)

var (
	addr = flag.String("listen", "/tmp/awsencryptionprovider.sock", "GRPC listen address")
)

func main() {
	flag.Parse()

	conn, err := connection.New(*addr)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}
	defer conn.Close()

	client := pb.NewKeyManagementServiceClient(conn)

	fmt.Println("Welcome to GRPC Client")
	fmt.Println("----------------------")

	ctx := context.Background()

	vReq := &pb.StatusRequest{}
	vRes, err := client.Status(ctx, vReq)
	if err != nil {
		log.Fatalf("Failed to get version: %v", err)
	}

	fmt.Println("Connected to GRPC Server", vRes.Version, vRes.Healthz, vRes.KeyId)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("encrypt <string>\ndecrypt <string>\n")
	for {
		fmt.Print("->")
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)

		splits := strings.SplitN(text, " ", 2)

		switch splits[0] {
		case "encrypt":
			eReq := &pb.EncryptRequest{Plaintext: []byte(splits[1])}
			res, err := client.Encrypt(ctx, eReq)
			if err != nil {
				log.Fatalf("Failed to encrypt: %v", err)
			}
			fmt.Println(base64.StdEncoding.EncodeToString(res.Ciphertext))
			fmt.Println("KeyId is: ", res.KeyId)
		case "decrypt":
			b, err := base64.StdEncoding.DecodeString(splits[1])
			if err != nil {
				log.Fatalf("Failed to decode: %v", err)
			}
			dReq := &pb.DecryptRequest{Ciphertext: b}
			res, err := client.Decrypt(ctx, dReq)
			if err != nil {
				log.Fatalf("Failed to encrypt: %v", err)
			}
			fmt.Println(string(res.Plaintext))
		}
	}
}
