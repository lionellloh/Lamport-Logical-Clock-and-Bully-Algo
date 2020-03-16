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
const MAX_NETWORK_DELAY float32 = 2
const TOTAL_PROCESSES int = NUMBER_OF_CLIENTS + 1

type server struct {
	pid int
	channel chan message
	clientArray []client
	logicalTS   int
	vectorClock [TOTAL_PROCESSES] int
}

type message struct {
	senderName    string
	receiverName string
	messageString string
	//logicalTS     int
	vectorClock [TOTAL_PROCESSES] int
}

type client struct {
	pid int
	name          string
	clientChannel chan message
	server        server
	//logicalTS     int
	readyChannel  chan int
	killChan chan int
	vectorClock [TOTAL_PROCESSES] int
}

func main() {
	//Test Merge()
	//v1 := [TOTAL_PROCESSES] int {2,3,1,4,5,7}
	//v2 := [TOTAL_PROCESSES] int {3,9,6,4,3,1}
	//fmt.Println(merge(v1, v2, 4))
	s := NewServer(0)

	for i := 1; i <= NUMBER_OF_CLIENTS; i++ {

		c := NewClient(i, fmt.Sprintf("Client %d", i), *s)
		c.registerWithServer(*s)
		s.clientArray = append(s.clientArray, *c)


	}
	var allMessages chan message =  make(chan message, NUMBER_OF_CLIENTS * NUMBER_OF_CLIENTS * NUMBER_OF_MESSAGES_TO_SEND*10)

	for _, c := range s.clientArray {
		fmt.Println(c.name)
		go c.timePing()
		go c.pingAndListen(allMessages)

	}

	s.listen(allMessages)
	fmt.Println(len(allMessages))
	messageArray := []message{}

	for _, client := range s.clientArray {
		client.killChan <- 1
	}

	close(allMessages)
	for message := range allMessages {
		messageArray = append(messageArray, message)
	}

	sort.Slice(messageArray, func(i, j int) bool {
		return messageArray[i].vectorClock < messageArray[j].vectorClock
	})


	fmt.Println("=========================")
	fmt.Println("        Total Order      ")
	fmt.Println("=========================")
	for i, m := range messageArray {
		fmt.Printf("%d: |Timestamp: %d | Sent from: %s | Sent to: %s | Message: %s \n", i, m.vectorClock, m.senderName, m.receiverName, m.messageString)
	}

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



func NewServer(pid int) *server {
	channel := make(chan message)
	clientArray := []client{}
	vectorClock := [TOTAL_PROCESSES] int {}

	s := server{pid, channel, clientArray, 0, vectorClock}

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

func (s server) listen(allMessages chan message) {
	//wg := sync.WaitGroup{}
	var numChannelsPinging int = NUMBER_OF_CLIENTS
	for {
		msg := <-s.channel
		allMessages <- msg
		s.vectorClock = merge(s.vectorClock, msg.vectorClock, s.pid)
		fmt.Printf("[TS: %d] Server: Received <%s> from client <%s> \n", s.logicalTS, msg.messageString, msg.senderName)

		s.vectorClock[s.pid] += 1
		//Server broadcasting
		for _, client := range s.clientArray {
			if client.name == msg.senderName {
				continue
			} else {
				newMessage := message{"Server", client.name,fmt.Sprintf("[Forwarded] %s", msg.messageString), s.vectorClock}
				//wg.Add(1)
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



func (c client) pingAndListen(allMessages chan message) {
	var clientMessage message

	for {
		select {

		case broadcastedMessage := <-c.clientChannel:
			allMessages <- broadcastedMessage
			//c.logicalTS = Max(c.logicalTS, broadcastedMessage.logicalTS) + 1
			c.vectorClock = merge(c.vectorClock, broadcastedMessage.vectorClock, c.pid)
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
