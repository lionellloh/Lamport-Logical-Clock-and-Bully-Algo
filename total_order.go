package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {

	s := NewServer()

	for i := 1; i <= 5; i++ {
		c := NewClient(fmt.Sprintf("Client %d", i), *s)
		c.registerWithServer(*s)
		s.clientArray = append(s.clientArray, *c)

	}

	for _, c := range s.clientArray {
		go c.listen()
	}

	for _, c := range s.clientArray {
		go c.ping()
	}

	s.listen()

}

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
	const MAX_DELAY float32 = 5
	//random time delay
	var numSeconds float32 = rand.Float32() * MAX_DELAY
	fmt.Printf("Simulated Network Latency %f seconds \n", numSeconds)
	time.Sleep(time.Duration(numSeconds) * time.Second)

	clientChannel <- msg
}

func (s server) listen() {

	for {
		msg := <-s.channel
		s.logicalTS = Max(s.logicalTS, msg.logicalTS) + 1
		fmt.Printf("[TS: %d] Server: Received <%s> from client <%s> \n", s.logicalTS, msg.messageString, msg.senderName)

		//advance the timestamp of the msg
		msg.logicalTS = s.logicalTS

		//Server broadcasting
		for _, client := range s.clientArray {
			if client.name == msg.senderName {
				continue
			} else {
				go sendMessage(client.clientChannel, msg)
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
	c := client{name, clientChannel, s, 0}
	return &c

}

func (c client) registerWithServer(s server) {
	c.server = s
	s.clientArray = s.addClient(c)
}

func (c client) ping() {
	var clientMessage message
	for {
		time.Sleep(2 * time.Second)
		clientMessage = message{c.name, fmt.Sprintf("Hello, I am %s", c.name), c.logicalTS}
		c.server.channel <- clientMessage
	}

}

func (c client) listen() {
	for {
		broadCastedMessage := <-c.clientChannel
		//Modify Logical Timestamp
		c.logicalTS = Max(c.logicalTS, broadCastedMessage.logicalTS) + 1
		fmt.Printf("[TS: %d] Client %s: Received '%s' from server \n", c.logicalTS, c.name, broadCastedMessage.messageString)
	}

}
