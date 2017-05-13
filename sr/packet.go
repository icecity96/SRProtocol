package sr

import (
	"errors"
	"time"
	"fmt"
	"log"
)

type Packet struct {
	SeqAvaility	bool `json:"seq_availity"`
	AckAvaility	bool `json:"ack_availity"`
	GBN		bool `json:"gbn"`
	SeqNo	int	`json:"seq_no"`
	AckNo 	int	`json:"ack_no"`
	Data 	[]byte	`json:"data"`
}

type SequenceNumber struct {
	Content 	Packet
	SequenceNumber int
	Sent		bool
	Acknowledge	bool
	TimerOut	*time.Timer
}

type Queue struct {
	contents 	[]SequenceNumber
	windowSize				int
	baseIndex				int
	nextSequenceNumberIndex	int
}

func NewQueue(windowSize int) *Queue {
	queue := new(Queue)

	queue.contents = make([]SequenceNumber,windowSize)
	queue.windowSize = windowSize
	queue.baseIndex = 0
	queue.nextSequenceNumberIndex = 0

	for i := range queue.contents {
		queue.contents[i].SequenceNumber = i
	}

	return queue
}

func (q *Queue) OldestUnacknowledgedSequenceNumber() (*SequenceNumber, error) {
	sequenceNumber := &q.contents[q.baseIndex]

	if !sequenceNumber.Sent {
		return nil, errors.New("Oldest sequence number has not been sent.")
	}

	return sequenceNumber, nil
}

func (q *Queue) Send() (int, error) {
	if q.Full() {
		return 0, errors.New("Window is full.")
	}
	q.contents[q.nextSequenceNumberIndex].Sent = true
	q.nextSequenceNumberIndex++

	if q.nextSequenceNumberIndex >= cap(q.contents) {
		numberToAdd := cap(q.contents)

		for i := numberToAdd; i <= 2 * numberToAdd; i++ {
			newSequenceNumber := SequenceNumber{SequenceNumber:i, Sent:false, Acknowledge:false}
			q.contents = append(q.contents, newSequenceNumber)
		}
	}

	return q.nextSequenceNumberIndex-1, nil
}

func (q *Queue) Full() bool {
	return q.FreeSpace() == 0
}

// Empty returns true if the queue is empty (i.e. no packets sent)
func (q *Queue) Empty() bool {
	return q.FreeSpace() == q.windowSize
}

// FreeSpace returns the number of available slots in the window
func (q *Queue) FreeSpace() int {
	return q.windowSize - (q.nextSequenceNumberIndex - q.baseIndex)
}

// MarkAckowledged marks the given sequence number as acknowledged if it is in the window
func (q *Queue) MarkAcknowledged(sequenceNumber int,gbn bool) (int,error) {
	if q.Empty() {
		return 0,errors.New("Window is empty")
	}
	if gbn {	// GBN
		var count int
		for i := q.baseIndex; i < q.nextSequenceNumberIndex; i++ {
			if q.contents[i].SequenceNumber < sequenceNumber {
				count++
				q.contents[i].Acknowledge = true
				q.contents[i].TimerOut.Stop()
				q.slideWindow()
			}
			return count,nil
		}
	} else {	// SR
		for i := q.baseIndex; i < q.nextSequenceNumberIndex; i++ {
			if i == sequenceNumber {
				q.contents[i].Acknowledge = true
				q.contents[i].TimerOut.Stop()
				log.Printf("SR: PACKET #%d Timer stop",i)
				if i == q.baseIndex {
					q.slideWindow()
				}
				return 1,nil
			}
		}
	}

	return 0,fmt.Errorf("Sequence number %d not found in window.", sequenceNumber)
}

// slideWindow slides the window until the first unacknowledged sequence number
func (q *Queue) slideWindow() {
	for q.nextSequenceNumberIndex > q.baseIndex && q.contents[q.baseIndex].Acknowledge == true {
		q.baseIndex++
	}
}
