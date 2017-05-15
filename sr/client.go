package sr

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"time"
)

type Client struct {
	queue   *Queue
	Conn    *net.UDPConn
	waiting chan int
}

func NewClient(windowSize int, serverAddr *net.UDPAddr) *Client {
	localAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, _ := net.DialUDP("udp", localAddr, serverAddr)
	return &Client{NewQueue(windowSize), conn, make(chan int, windowSize)}
}

func (c *Client) Send(sendLoss, ackLoss bool) (int, error) {
	sequenceNumber, err := c.queue.Send()
	if err != nil {
		<-c.waiting
		return c.Send(sendLoss, ackLoss)
	}
	return c.sendSequenceNumber(sequenceNumber, sendLoss, ackLoss)
}

func (c *Client) sendSequenceNumber(sequenceNumber int, senderLose, acknowledgementLose bool) (int, error) {
	timerOut := time.AfterFunc(5*time.Second, func() {
		timeoutTriggered(sequenceNumber)

		if !senderLose && acknowledgementLose {
			c.timeout(sequenceNumber, false)
		} else {
			c.timeout(sequenceNumber, acknowledgementLose)
		}
	})

	packet := Packet{SeqAvaility: true, AckAvaility: false, SeqNo: sequenceNumber, AckNo: 0, Data: []byte("HELLO SERVER")}
	c.queue.contents[sequenceNumber].Content = packet
	c.queue.contents[sequenceNumber].TimerOut = timerOut
	// Don't actually send the packet if we're supposed to "lose" it
	if !senderLose {
		go func() {
			b, _ := json.Marshal(c.queue.contents[sequenceNumber].Content)
			c.Conn.Write(b)
		}()
	}

	return sequenceNumber, nil
}

func (c *Client) Receive() (Packet, int, error) {
	b := make([]byte, 1024)
	n, err := c.Conn.Read(b)
	if err != nil {
		return Packet{}, 0, errors.New("READ UDP ERROR")
	}
	var packet Packet
	err = json.Unmarshal(b[:n], &packet)
	if err != nil {
		return Packet{}, 0, errors.New("JSON ERROR")
	}

	var count int
	if packet.AckAvaility {
		if count, err = c.queue.MarkAcknowledged(packet.AckNo, packet.GBN); err != nil {
			return Packet{}, 0, err
		}

		for i := 0; i < c.queue.FreeSpace(); i++ {
			select {
			case c.waiting <- 1:
			default:
			}
		}
	}
	return packet, count, nil
}

func (c *Client) timeout(sequenceNumber int, acknowledgementLose bool) (int, error) {
	return c.sendSequenceNumber(sequenceNumber, false, acknowledgementLose)
}

func timeoutTriggered(sequenceNumber int) {
	log.Printf("Sender timeout triggered for Packet #%d, resending...", sequenceNumber)
}
