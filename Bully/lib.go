package main

import (
	"fmt"
	"time"
)

// Timeout is the maximum time sending a message should take
const Timeout time.Duration = 2 * time.Second

// Message represents a message sent among the instances
type Message struct {
	Type   MessageCat
	Sender *Instance
}

// MessageCat is an enumerated type that represents the various message types
type MessageCat string

const (
	MiscMsg           MessageCat = "Others"
	Register          MessageCat = "Register"
	ElectReq          MessageCat = "Election Request"
	ElectLoss         MessageCat = "Election Rejection"
	ElectAnnouncement MessageCat = "Election Announcement"
	SendTimeout       MessageCat = "Send Timeout"
)

// ElectionState represents the election process on each instance
type ElectionState struct {
	Coordinator   *Instance
	State         ElectionStage
	AwaitingReply map[*Instance]*Instance
}

// ElectionStage represents the state of the election process on each instance
type ElectionStage string

const (
	Running      ElectionStage = "Election in Progress"
	LostElection ElectionStage = "Election in Progress, Lost Election"
	NotOngoing   ElectionStage = "Election not in Progress"
)



// Instance represent a instance in a distributed system
type Instance struct {
	ID       int
	InChan   chan Message
	Peers    map[*Instance]*Instance
	elecProc ElectionState
}

// ConstructNewNode returns a new Instance
func ConstructNewNode(ID int) *Instance {
	instance := &Instance{
		ID,
		make(chan Message),
		make(map[*Instance]*Instance, 0),
		ElectionState{
			nil,
			NotOngoing,
			make(map[*Instance]*Instance),
		},
	}
	// return
	return instance
}

func (n *Instance) start(killChan chan bool, peers []*Instance) {
	fmt.Println("Instance", n.ID, "is now alive")
	// register with all the peers
	for _, instance := range peers {
		n.Peers[instance] = instance
	}
	regMsg := Message{Register, n}
	for _, peer := range n.Peers {
		n.SendMessage(peer, regMsg)
	}
	// start an election after all peers have registered
	go func() {
		time.Sleep(Timeout)
		fmt.Println("Instance", n.ID, "-  starting election")
		n.startElection()
	}()
	// main loop
	for {
		select {
		case msg := <-n.InChan:
			n.handle(msg)
		case <-killChan:
			return
		}
	}
}

func (n *Instance) startElection() {
	// send ElectReq message to all higher ID peers
	fmt.Println("Instance", n.ID, "runs for election")
	n.elecProc.State = Running
	msg := Message{ElectReq, n}
	numSends := 0
	for _, peer := range n.Peers {
		if peer.ID > n.ID {
			n.SendMessage(peer, msg)
			n.elecProc.AwaitingReply[peer] = peer
			numSends++
		}
	}
	// if it doesn't send to any peers, then it wins the election
	if numSends == 0 {
		fmt.Println("Instance", n.ID, "has no higher ID peers")
		n.onElected()
	}
}

func (n *Instance) onElected() {
	fmt.Println("Instance", n.ID, "is elected as the new coordinator")
	n.elecProc.State = NotOngoing
	n.elecProc.Coordinator = n
	msg := Message{ElectAnnouncement, n}
	for _, peer := range n.Peers {
		if peer.ID < n.ID {
			n.SendMessage(peer, msg)
		}
	}
}


func (n *Instance) SendMessage(receiver *Instance, msg Message) {
	fmt.Println("Instance", n.ID, "sending message", msg, "to instance", receiver.ID)
	go func() {
		timeoutCh := make(chan bool, 1)
		go func() {
			time.Sleep(Timeout)
			timeoutCh <- true
		}()
		select {
		case receiver.InChan <- msg:
			return
		case <-timeoutCh:
			n.InChan <- Message{SendTimeout, receiver}
		}
		fmt.Println("Instance", n.ID, "finished sending message", msg, "to instance", receiver.ID)
	}()
}

func (n *Instance) handle(msg Message) {
	fmt.Println("Instance", n.ID, "received message", msg, "from instance", msg.Sender.ID)
	switch msg.Type {
	case MiscMsg:
		// No need to do anything
	case Register:
		// add sender to peer list
		n.Peers[msg.Sender] = msg.Sender
		fmt.Println("Instance", n.ID, "added instance", msg.Sender.ID, "to peer list")
	case ElectReq:
		// reply with rejection
		reply := Message{ElectLoss, n}
		n.SendMessage(msg.Sender, reply)
		// start election if election not already ongoing
		if n.elecProc.State == NotOngoing {
			n.startElection()
		}

	case ElectAnnouncement:
		// end election, clear list
		fmt.Println("Instance", n.ID, "ends election as it is notified of the new coordinator")
		n.elecProc.State = NotOngoing
		n.elecProc.AwaitingReply = make(map[*Instance]*Instance)
		n.elecProc.Coordinator = msg.Sender

	case ElectLoss:
		// stop running for election, clear list
		if n.elecProc.State == Running {
			fmt.Println("Instance", n.ID, "loses election to", msg.Sender.ID, "and stops running for election")
			n.elecProc.State = LostElection
			n.elecProc.AwaitingReply = make(map[*Instance]*Instance)
		}

	case SendTimeout:
		// remove instance from peer list
		fmt.Println("Instance", n.ID, "realised that instance", msg.Sender.ID, "is dead, removes it from peer list")
		delete(n.Peers, msg.Sender)
		switch n.elecProc.State {
		case NotOngoing:
			if msg.Sender == n.elecProc.Coordinator {
				// elect new coordinator
				fmt.Println("Instance", n.ID, " - realised coordinator is dead")
				n.startElection()
			}
		case Running:
			// check if the timeout is from a instance which the election process is waiting for
			for instance := range n.elecProc.AwaitingReply {
				if instance == msg.Sender {
					delete(n.elecProc.AwaitingReply, instance)
					// check if election process is still waiting for any instance
					if len(n.elecProc.AwaitingReply) == 0 {
						fmt.Println("Instance", n.ID, " - finished waiting for election replies")
						n.onElected()
					}
				}
			}
		}
	}
}



