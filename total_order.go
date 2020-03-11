package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const NUMBER_OF_MESSAGES_TO_SEND int = 5
const NUMBER_OF_CLIENTS int = 5
const CLIENT_PING_INTERVAL float32 = 2
const MAX_NETWORK_DELAY float32 = 5

type server struct {
	channel chan message
	//clientChannels []chan message
	clientArray []client
	logicalTS int
}

type message struct {
	senderName    string
	messageString string
	logicalTS int
}

type client struct {
	name          string
	clientChannel chan message
	server        server
	logicalTS int
	readyChannel chan int
	//lock sync.Mutex
	//logicalTS logTS
	//logicalTSChan chan int
	//messagesReceived []message
	//messagesReceived chan message
}

type logTS struct {
	numTS int
	mux sync.Mutex
}

func main() {

	s := NewServer()

	for i := 1; i <= NUMBER_OF_CLIENTS; i++ {

		c := NewClient(fmt.Sprintf("Client %d", i), *s)

		c.registerWithServer(*s)
		s.clientArray = append(s.clientArray, *c)

	}


	for _, c := range s.clientArray {
		fmt.Println(c.name)

		go c.timePing()
		go c.pingAndListen()
	}

	s.listen()

}



func NewServer() *server {
	channel := make(chan message)
	clientArray := []client{}

	s := server{channel, clientArray, 0}

	return &s
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func sendMessage(clientChannel chan message, msg message) {

	//random time delay
	var numSeconds float32 = rand.Float32() * MAX_NETWORK_DELAY
	//fmt.Printf("Simulated Network Latency %f seconds \n", numSeconds)
	time.Sleep(time.Duration(numSeconds) * time.Second)

	clientChannel <- msg
}

func (s server) listen() {

	for {
		msg := <-s.channel
		s.logicalTS = Max(s.logicalTS, msg.logicalTS) + 1
		fmt.Printf("[TS: %d] Server: Received <%s> from client <%s> \n", s.logicalTS, msg.messageString, msg.senderName)

		//Server broadcasting
		for _, client := range s.clientArray {
			if client.name == msg.senderName {
				continue
			} else {
				newMessage := message{"Server", msg.messageString, s.logicalTS}
				go sendMessage(client.clientChannel, newMessage)
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
func NewClient(name string, s server) *client {

	clientChannel := make(chan message)
	readyChannel := make(chan int)
	c := client{name, clientChannel, s, 0, readyChannel}
	return &c

}



func (c *client) registerWithServer(s server) {
	c.server = s
	s.clientArray = s.addClient(*c)
}

func (c client) timePing(){
	for i:=1; i <= NUMBER_OF_MESSAGES_TO_SEND; i++ {
		time.Sleep(time.Duration(CLIENT_PING_INTERVAL) * time.Second)
		c.readyChannel <- i
	}
}

//func (c client) ping() {
//	const TIME_INTERVAL int = 2
//	var clientMessage message
//	for  i := 0; i < NUMBER_OF_MESSAGES_TO_SEND; i ++{
//		time.Sleep(time.Duration(TIME_INTERVAL) * time.Second)
//		//selfLTS := <- c.logicalTSChan
//		//
//		//c.logicalTSChan <- selfLTS + 1
//		c.lock.Lock()
//		//c.logicalTS.mux.Lock()
//		//lts := c.logicalTS.numTS
//		//c.logicalTS.numTS = lts + 1
//
//		lts := <- c.logicalTSChan + 1
//		fmt.Printf( "ping: %d %s \n",  lts, c.name)
//		//c.logicalTS.mux.Unlock()
//
//
//		clientMessage = message{c.name, fmt.Sprintf("Hello, I am %s", c.name), lts }
//		c.logicalTSChan <- lts
//		c.lock.Unlock()
//		c.server.channel <- clientMessage
//
//	}
//
//	fmt.Printf("%s has finished sending %d messages \n", c.name, NUMBER_OF_MESSAGES_TO_SEND)
//	//time.Sleep(5 * time.Second)
//	fmt.Println(len(c.messagesReceived))
//	//for i, msg := range c.messagesReceived {
//	//	fmt.Printf("hehehe %dth Message: [TS: %d] Message: %s", i, msg.logicalTS, msg.messageString)
//	//}
//}

//func (c client) listen() {
//
//	for {
//		c.lock.Lock()
//		broadCastedMessage := <-c.clientChannel
//		//lts := <- c.logicalTSChan
//		//c.messagesReceived = append(c.messagesReceived, broadCastedMessage)
//		//Modify Logical Timestamp
//		//c.logicalTS.mux.Lock()
//		lts := Max(<- c.logicalTSChan, broadCastedMessage.logicalTS) + 1
//		fmt.Println(broadCastedMessage.messageString, broadCastedMessage.senderName)
//		//c.logicalTS.numTS = lts
//
//		fmt.Printf( "listening: %d %s \n", lts, c.name)
//
//		c.logicalTSChan <- lts
//		c.lock.Unlock()
//	}
//
//}


func (c client) pingAndListen(){

	for {
		select {
			case broadcastedMessage := <- c.clientChannel:
				c.logicalTS = Max(c.logicalTS, broadcastedMessage.logicalTS) + 1
				fmt.Printf("[TS: %d] %s: Received '%s' from %s \n", c.logicalTS, c.name, broadcastedMessage.messageString, broadcastedMessage.senderName)
			case messageNo := <- c.readyChannel:
				fmt.Printf("%s is pinging now at TS: %d \n", c.name, c.logicalTS)
				c.logicalTS +=1
				clientMessage := message{c.name, fmt.Sprintf("Hello %d from %s", messageNo, c.name), c.logicalTS }
				c.server.channel <- clientMessage
		}
	}
}