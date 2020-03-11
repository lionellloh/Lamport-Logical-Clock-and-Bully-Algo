package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const NUMBER_OF_MESSAGES_TO_SEND int = 3
const NUMBER_OF_CLIENTS int = 5
const CLIENT_PING_INTERVAL float32 = 2
const MAX_NETWORK_DELAY float32 = 2

type server struct {
	channel chan message
	//clientChannels []chan message
	clientArray []client
	logicalTS   int
}

type message struct {
	senderName    string
	messageString string
	logicalTS     int
}

type client struct {
	name          string
	clientChannel chan message
	server        server
	logicalTS     int
	readyChannel  chan int
}

type logTS struct {
	numTS int
	mux   sync.Mutex
}

func main() {

	var wg sync.WaitGroup
	wg.Add(1)

	s := NewServer()

	for i := 1; i <= NUMBER_OF_CLIENTS; i++ {

		c := NewClient(fmt.Sprintf("Client %d", i), *s)
		c.registerWithServer(*s)
		s.clientArray = append(s.clientArray, *c)

	}
	var allMessages chan message =  make(chan message, NUMBER_OF_CLIENTS * NUMBER_OF_CLIENTS * NUMBER_OF_MESSAGES_TO_SEND)

	for _, c := range s.clientArray {
		fmt.Println(c.name)
		go c.timePing()
		go c.pingAndListen(allMessages)

	}

	s.listen(allMessages)

	fmt.Println("ALL DONE!")
	fmt.Println(len(allMessages))
	messageArray := []message{}
	for message := range allMessages {
		messageArray = append(messageArray, message)
	}

	fmt.Println(len(messageArray))

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
	fmt.Println("send message", msg.messageString, msg.senderName)
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
		//allMessages <- msg
		s.logicalTS = Max(s.logicalTS, msg.logicalTS) + 1
		fmt.Printf("[TS: %d] Server: Received <%s> from client <%s> \n", s.logicalTS, msg.messageString, msg.senderName)

		//Server broadcasting
		for _, client := range s.clientArray {
			if client.name == msg.senderName {
				continue
			} else {
				newMessage := message{"Server", fmt.Sprintf("[Forwarded] %s", msg.messageString), s.logicalTS}
				//wg.Add(1)
				go sendMessage(client.clientChannel, newMessage)
			}

		}

		if msg.messageString == "LAST" {
			fmt.Printf("LAST from %s \n", msg.senderName)
			numChannelsPinging -= 1
			fmt.Println("NUM CHANNELS: ", numChannelsPinging)

			if numChannelsPinging == 0 {
				//wg.Wait()
				time.Sleep(time.Duration(MAX_NETWORK_DELAY) * time.Second)
				//for _, client := range s.clientArray {
				//	fmt.Printf("Server closing %s's channel \n", client.name)
					//time.Sleep(10 * time.Second)
					//close(client.clientChannel)
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

func (c client) timePing() {

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
			//allMessages <- broadcastedMessage
			c.logicalTS = Max(c.logicalTS, broadcastedMessage.logicalTS) + 1
			fmt.Printf("[TS: %d] %s: Received '%s' from %s \n", c.logicalTS, c.name, broadcastedMessage.messageString, broadcastedMessage.senderName)
		case messageNo := <-c.readyChannel:
			fmt.Printf("%s is pinging now at TS: %d for message %d \n", c.name, c.logicalTS, messageNo)
			c.logicalTS += 1
			if messageNo == NUMBER_OF_MESSAGES_TO_SEND {
				clientMessage = message{c.name, "LAST", c.logicalTS}
			} else {
				clientMessage = message{c.name, fmt.Sprintf("Hello %d from %s", messageNo, c.name), c.logicalTS}
			}
			c.server.channel <- clientMessage
		default:

		}
	}
}
