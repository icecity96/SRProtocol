package sr

import (
	"net"
	"encoding/json"
	"log"
)

type Server struct {
	queue   *Queue
	Conn    *net.UDPConn
	waiting chan int
	gbn     bool
}

func NewServer(windowSize int, serverAddr *net.UDPAddr) *Server {
	conn, _:= net.ListenUDP("udp", serverAddr)
	var gbn bool
	if windowSize == 1 {
		gbn = true
	} else {
		gbn = false
	}

	queue := NewQueue(windowSize)
	queue.nextSequenceNumberIndex = windowSize

	return &Server{queue,conn,make(chan int, windowSize),gbn}
}

func (s *Server) Recieve() {
	b := make([]byte,1024)
	n, remoterAddr,_ := s.Conn.ReadFromUDP(b)
	if n == 0 {
		return
	}
	var packet Packet
	json.Unmarshal(b[0:n],&packet)
	packet.GBN = s.gbn
	packet.AckAvaility = true
	if packet.GBN {
		if packet.SeqNo == s.queue.baseIndex {
			packet.AckNo = packet.SeqNo + 1
			log.Printf("GBN : SERVER RECEIVE PACKET %d, ASK FOR %d",packet.SeqNo,packet.AckNo)
			s.queue.baseIndex++
			s.queue.nextSequenceNumberIndex++
			packet.SeqAvaility = false
			b,_ = json.Marshal(packet)
			s.Conn.WriteTo(b,remoterAddr)
		} else {
			log.Printf("GBN: DISCARD UNORDERED PACKET %d",packet.SeqNo)
		}
	} else {
		if s.queue.nextSequenceNumberIndex <= packet.SeqNo {
			log.Printf("SR: DISCARD OUT RANGE PACKET %d",packet.SeqNo)
		} else if s.queue.contents[packet.SeqNo].Acknowledge {
			log.Printf("SR: DISCARD DUPLICATE PACKET %d",packet.SeqNo)
		} else if s.queue.baseIndex == packet.SeqNo {
			s.queue.contents[s.queue.baseIndex].Acknowledge = true
			s.queue.slideWindow()
			s.queue.nextSequenceNumberIndex += s.queue.windowSize - s.queue.baseIndex
			if s.queue.nextSequenceNumberIndex >= cap(s.queue.contents) {
				numberToAdd := cap(s.queue.contents)
				for i := cap(s.queue.contents); i <= numberToAdd; i++ {
					newSequenceNumber := SequenceNumber{SequenceNumber:i, Sent:false, Acknowledge:false}
					s.queue.contents = append(s.queue.contents, newSequenceNumber)
				}
			}
			packet.AckNo = packet.SeqNo
			packet.SeqAvaility = false
			b,_ = json.Marshal(packet)
			s.Conn.WriteTo(b,remoterAddr)
			log.Printf("SR: GET PACKET %d, AND SLIDE WINDOW TO %d",packet.SeqNo,s.queue.baseIndex)
		} else {
			s.queue.contents[packet.SeqNo].Acknowledge = true
			s.queue.contents[packet.SeqNo].Content = packet
			packet.AckNo = packet.SeqNo
			packet.SeqAvaility = false
			b,_ = json.Marshal(packet)
			s.Conn.WriteTo(b,remoterAddr)
			log.Printf("SR: STORE PACKET %d",packet.SeqNo)
		}
	}
}

