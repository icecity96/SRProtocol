package main

import (
	"SRProtocol/sr"
	"flag"
	"log"
	"net"
	"time"
)

func main() {

	log.SetFlags(log.Lmicroseconds)

	packetSequence := flag.String("packet-sequence", "__S_", "The sequence of packets to send. '_' no losses, 'A' ACK loss, 'S', sender loss, 'B' both lost")
	timeBetweenPackets := flag.Duration("packet-time", 50*time.Millisecond, "Amount of time waited after each packet is sent.")
	senderWindowSize := flag.Int("sender-window-size", 8, "Window size for the selective repeat protocol")
	reciverWindowSize := flag.Int("reciver-window-size", 8, "Window size for the selective repeat protocol")

	flag.Parse()

	packetLoss := parsePacketSequence(*packetSequence)
	ServerAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	client := sr.NewClient(*senderWindowSize, ServerAddr)
	server := sr.NewServer(*reciverWindowSize, ServerAddr)
	defer client.Conn.Close()
	defer server.Conn.Close()
	k := make(chan int, len(*packetSequence))
	go receiveHandler(client, "CLIENT", k)
	go receiveHandler2(server, "SERVER", nil)
	go func() {
		for _, v := range packetLoss {
			go send(client, v.Sender, v.Acknowledgment)
			time.Sleep(*timeBetweenPackets)
		}
	}()

	for i := 0; i < len(*packetSequence); i++ {
		<-k
	}
}

type PacketLoss struct {
	Sender         bool
	Acknowledgment bool
}

func parsePacketSequence(packetSequence string) []PacketLoss {
	loss := make([]PacketLoss, len(packetSequence))

	for i, v := range packetSequence {
		packetLoss := PacketLoss{false, false}

		switch v {
		case 'A', 'a':
			packetLoss.Acknowledgment = true
		case 'S', 's':
			packetLoss.Sender = true
		case 'B', 'b':
			packetLoss.Sender = true
			packetLoss.Acknowledgment = true
		}

		loss[i] = packetLoss
	}

	return loss
}

func send(c *sr.Client, senderLoss, acknowledgementLoss bool) {
	if sequenceNumber, err := c.Send(senderLoss, acknowledgementLoss); err != nil {
		log.Println("Error - ", err)
	} else {
		log.Printf("Sender sent packet with sequence number %d\n", sequenceNumber)
	}
}

func receiveHandler(c *sr.Client, name string, receivedACK chan int) {
	for {
		packet, count, err := c.Receive()

		if packet.AckAvaility {
			for i := 0; i < count; i++ {
				receivedACK <- 1
			}
		}

		if err != nil {
			log.Println("Error - ", err)
		} else {
			log.Printf("%s received ACK: %v\n", name, packet.AckNo)
		}
	}
}

func receiveHandler2(c *sr.Server, name string, receivedACK chan int) {
	for {
		c.Recieve()
	}
}
