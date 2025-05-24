package main

import (
	"context"
	"fmt"
	"log"
	"time"

	nodepb "github.com/blackhole/core/pkg/plugins/node/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the node plugin via Unix socket
	conn, err := grpc.Dial("unix:///tmp/blackhole/plugins/node.sock",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to node plugin: %v", err)
	}
	defer conn.Close()

	// Create client
	client := nodepb.NewNodeServiceClient(conn)
	ctx := context.Background()

	// Get node information
	fmt.Println("=== Node Information ===")
	info, err := client.GetInfo(ctx, &nodepb.GetInfoRequest{})
	if err != nil {
		log.Fatalf("Failed to get node info: %v", err)
	}

	fmt.Printf("Node ID: %s\n", info.Id)
	fmt.Printf("Addresses: %v\n", info.Addresses)
	fmt.Printf("Protocols: %v\n", info.Protocols)
	fmt.Printf("Agent Version: %s\n", info.AgentVersion)

	// Get node status
	fmt.Println("\n=== Node Status ===")
	status, err := client.GetStatus(ctx, &nodepb.GetStatusRequest{})
	if err != nil {
		log.Fatalf("Failed to get node status: %v", err)
	}

	fmt.Printf("Status: %s\n", status.Status)
	fmt.Printf("Connected Peers: %d\n", status.PeerCount)
	if status.Network != nil {
		fmt.Printf("Bytes Sent: %d\n", status.Network.BytesSent)
		fmt.Printf("Bytes Received: %d\n", status.Network.BytesReceived)
	}

	// List connected peers
	fmt.Println("\n=== Connected Peers ===")
	peers, err := client.ListPeers(ctx, &nodepb.ListPeersRequest{
		ConnectedOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to list peers: %v", err)
	}

	for _, peer := range peers.Peers {
		fmt.Printf("Peer ID: %s\n", peer.Id)
		fmt.Printf("  Addresses: %v\n", peer.Addresses)
		fmt.Printf("  Connected At: %s\n", time.Unix(peer.ConnectedAt, 0))
	}

	// Subscribe to a topic
	fmt.Println("\n=== Subscribing to 'example-topic' ===")
	stream, err := client.Subscribe(ctx, &nodepb.SubscribeRequest{
		Topic:       "example-topic",
		IncludeSelf: false,
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Start a goroutine to handle incoming messages
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Printf("Stream error: %v", err)
				return
			}

			fmt.Printf("\n[Message Received]\n")
			fmt.Printf("  From: %s\n", msg.PeerId)
			fmt.Printf("  Topic: %s\n", msg.Topic)
			fmt.Printf("  Data: %s\n", string(msg.Data))
			fmt.Printf("  Time: %s\n", time.Unix(msg.Timestamp, 0))
		}
	}()

	// Publish a message after a short delay
	time.Sleep(2 * time.Second)
	fmt.Println("\n=== Publishing Message ===")
	pubResp, err := client.Publish(ctx, &nodepb.PublishRequest{
		Topic: "example-topic",
		Data:  []byte("Hello from the example application!"),
		Headers: map[string]string{
			"app": "node-example",
			"version": "1.0.0",
		},
	})
	if err != nil {
		log.Printf("Failed to publish: %v", err)
	} else {
		fmt.Printf("Message published with ID: %s\n", pubResp.MessageId)
	}

	// Keep the program running to receive messages
	fmt.Println("\nPress Ctrl+C to exit...")
	select {}
}