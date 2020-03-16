package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

const NUMBER_OF_MESSAGES_TO_SEND int = 3
const NUMBER_OF_CLIENTS int = 5
const CLIENT_PING_INTERVAL float32 = 2
const MAX_NETWORK_DELAY float32 = 10
const TOTAL_PROCESSES int = NUMBER_OF_CLIENTS + 1

type server struct {
	pid int
	channel chan message
	clientArray []client
	vectorClock [TOTAL_PROCESSES] int
}

type message struct {
	senderName    string
	receiverName string
	messageString string
	vectorClock [TOTAL_PROCESSES] int
}

type client struct {
	pid int
	name          string
	clientChannel chan message
	server        server
	readyChannel  chan int
	killChan chan int
	vectorClock [TOTAL_PROCESSES] int
}

func main() {
	//Test Merge()
	//v1 := [TOTAL_PROCESSES] int {2,3,1,4,3,2}
	//v2 := [TOTAL_PROCESSES] int {3,9,6,4,3,5}
	//
	//fmt.Println(lessThan(v1, v2))
	//
	//
	//fmt.Println(merge(v1, v2, 4))
	s := NewServer(0)

	for i := 1; i <= NUMBER_OF_CLIENTS; i++ {

		c := NewClient(i, fmt.Sprintf("Client %d", i), *s)
		c.registerWithServer(*s)
		s.clientArray = append(s.clientArray, *c)


	}
	var allMessages chan message =  make(chan message, NUMBER_OF_CLIENTS * NUMBER_OF_CLIENTS * NUMBER_OF_MESSAGES_TO_SEND*10)
	//Store all potential instances of causality violations
	var potentialCV chan string = make(chan string, NUMBER_OF_CLIENTS * NUMBER_OF_CLIENTS * NUMBER_OF_MESSAGES_TO_SEND)
	for _, c := range s.clientArray {
		fmt.Println(c.name)
		go c.timePing()
		go c.pingAndListen(allMessages, potentialCV)

	}

	s.listen(allMessages, potentialCV)
	fmt.Println(len(allMessages))
	messageArray := []message{}

	for _, client := range s.clientArray {
		client.killChan <- 1
	}

	close(allMessages)
	close(potentialCV)
	for message := range allMessages {
		messageArray = append(messageArray, message)
	}

	sort.Slice(messageArray, func(i, j int) bool {
		return lessThan(messageArray[i].vectorClock, messageArray[j].vectorClock)
	})


	fmt.Println("=========================")
	fmt.Println("        Total Order      ")
	fmt.Println("=========================")
	for i, m := range messageArray {
		fmt.Printf("%d: |Timestamp: %d | Sent from: %s | Sent to: %s | Message: %s \n", i, m.vectorClock, m.senderName, m.receiverName, m.messageString)
	}

	pcvArray := [] string{}
	for pcv := range potentialCV {
		pcvArray = append(pcvArray, pcv)
	}


	fmt.Println("=================================================")
	fmt.Println("           Potential Causality Violation         ")
	fmt.Println("=================================================")

	for _, m := range pcvArray {
		fmt.Println(m)
	}

}

func lessThan(a1 [TOTAL_PROCESSES] int, a2 [TOTAL_PROCESSES] int ) bool{
	//Compare 2 vector clocks... If a1 is element wise <= than a2, return true
	for i, _ := range a1{
		if a1[i] > a2[i] {
			return false
		}
	}
	return true
}


func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}


func merge(vcReceiver [TOTAL_PROCESSES]int, vcSender [TOTAL_PROCESSES]int, receiverPID int) [TOTAL_PROCESSES]int {
	vcRet := [TOTAL_PROCESSES] int {}
	for i, _ := range vcReceiver {
		//vcRet = append(vcRet, Max(vcSender[i], vcReceiver[i]))
		vcRet[i] = Max(vcSender[i], vcReceiver[i])
	}

		vcRet[receiverPID] +=1
		return vcRet
}

func moreThan(vcReceiver [TOTAL_PROCESSES]int, vcSender [TOTAL_PROCESSES]int) bool {
	//If vcReciver > vcSender: return True. Else false.
	for i, _ := range vcReceiver {
		//vcRet = append(vcRet, Max(vcSender[i], vcReceiver[i]))
		if vcReceiver[i] <= vcSender[i]{
			return false
		}
 	}
 	return true
}


func NewServer(pid int) *server {
	channel := make(chan message)
	clientArray := []client{}
	vectorClock := [TOTAL_PROCESSES] int {}

	s := server{pid, channel, clientArray, vectorClock}

	return &s
}



func sendMessage(clientChannel chan message, msg message) {
	fmt.Println(msg.messageString, msg.senderName)
	//random time delay
	var numSeconds float32 = rand.Float32() * MAX_NETWORK_DELAY
	//fmt.Printf("Simulated Network Latency %f seconds \n", numSeconds)
	time.Sleep(time.Duration(numSeconds) * time.Second)
	//fmt.Println(msg.messageString)
	//defer wg.Done()
	clientChannel <- msg

	return
}

func (s server) listen(allMessages chan message, potentialCV chan string) {
	var numChannelsPinging int = NUMBER_OF_CLIENTS
	for {
		msg := <-s.channel

		s.vectorClock = merge(s.vectorClock, msg.vectorClock, s.pid)
		if moreThan(s.vectorClock, msg.vectorClock) {
			fmt.Println("POTENTIAL CAUSALITY VIOLATION")
			potentialCV <- fmt.Sprintf("From: %s| To: %s |Msg Vector Clock: %d| Local Vector Clock>: %d | Message Text: %s",
				msg.senderName, msg.receiverName, msg.vectorClock, s.vectorClock, msg.messageString)
		}
		msg.vectorClock = s.vectorClock
		allMessages <- msg
		fmt.Printf("[TS: %d] Server: Received <%s> from client <%s> \n", msg.vectorClock, msg.senderName)

		s.vectorClock[s.pid] += 1
		//Server broadcasting
		for _, client := range s.clientArray {
			if client.name == msg.senderName {
				continue
			} else {
				newMessage := message{"Server", client.name,fmt.Sprintf("[Forwarded] %s", msg.messageString), s.vectorClock}
				go sendMessage(client.clientChannel, newMessage)
			}

		}

		if msg.messageString == "LAST" {
			fmt.Printf("LAST from %s \n", msg.senderName)
			numChannelsPinging -= 1
			fmt.Println("NUM CHANNELS: ", numChannelsPinging)

			if numChannelsPinging == 0 {
				time.Sleep(time.Duration(MAX_NETWORK_DELAY) * time.Second)

				fmt.Println("Returning Server Routine")

				return
			}

		}
	}
}


func (s *server) addClient(c client) []client {
	fmt.Printf("Registering %s \n", c.name)
	s.clientArray = append(s.clientArray, c)
	return s.clientArray
}

// Constructor for client
func NewClient( pid int, name string, s server) *client {

	clientChannel := make(chan message)
	readyChannel := make(chan int)
	killChan := make(chan int)
	vectorClock := [TOTAL_PROCESSES] int {}
	c := client{pid, name, clientChannel, s, readyChannel, killChan, vectorClock}
	return &c

}


func (c *client) registerWithServer(s server) {
	c.server = s
	s.clientArray = s.addClient(*c)
}

func (c client) timePing() {
	//A go routine that keeps time so as to alert the client routine to ping without blocking the client from listening
	for i := 1; i <= NUMBER_OF_MESSAGES_TO_SEND; i++ {
		time.Sleep(time.Duration(CLIENT_PING_INTERVAL) * time.Second)
		c.readyChannel <- i
	}
	//close(c.readyChannel)
	return
}



func (c client) pingAndListen(allMessages chan message, potentialCV chan string) {
	var clientMessage message

	for {
		select {

		case broadcastedMessage := <-c.clientChannel:

			//c.logicalTS = Max(c.logicalTS, broadcastedMessage.logicalTS) + 1
			c.vectorClock = merge(c.vectorClock, broadcastedMessage.vectorClock, c.pid)
			if moreThan(c.vectorClock, broadcastedMessage.vectorClock){
				fmt.Println("DETECTED POTENTIAL CAUSALITY VIOLATION")
				potentialCV <- fmt.Sprintf("From: %s| To: %s |Msg Vector Clock: %d| Local Vector Clock: %d | Message Text: %s",
					broadcastedMessage.senderName, broadcastedMessage.receiverName, broadcastedMessage.vectorClock, c.vectorClock, broadcastedMessage.messageString)
			}
			broadcastedMessage.vectorClock = c.vectorClock
			allMessages <- broadcastedMessage

			fmt.Printf("[TS: %d] %s: Received '%s' from %s \n", c.vectorClock, c.name, broadcastedMessage.messageString, broadcastedMessage.senderName)
		case messageNo := <-c.readyChannel:
			c.vectorClock[c.pid] +=1
			fmt.Printf("%s is pinging now at TS: %d for message %d \n", c.name, c.vectorClock, messageNo)
			if messageNo == NUMBER_OF_MESSAGES_TO_SEND {
				clientMessage = message{c.name, "Server", "LAST", c.vectorClock}
			} else {
				clientMessage = message{c.name, "Server", fmt.Sprintf("Hello %d from %s", messageNo, c.name), c.vectorClock}
			}
			c.server.channel <- clientMessage

		case <- c.killChan:
			return
		default:

		}
	}
}
