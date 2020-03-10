package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {


	s := NewServer()
	clients := []client {}


	for i := 1;  i<=5; i++ {
		c := NewClient(fmt.Sprintf("Client %d", i), *s)
		c.registerWithServer(*s)
		clients = append(clients, *c)
		s.clientChannels = append(s.clientChannels, c.clientChannel)

	}

	for _, c := range clients {
		go c.listen()
	}

	for _, c := range clients {
		go c.ping()
	}

	s.listen()

}

type server struct {
	channel chan message
	//clientChannels []chan message
	clients []client

}

type message struct {
	senderName string
	messageString string
}

func NewServer() *server{
	channel := make(chan message)
	clientArray := []client {}

	s := server{channel, clientArray}

	return &s
}

func (s server) listen() {
	for {
		//random time delay
		var max float32 = 5
		var numSeconds float32 = rand.Float32() * max
		fmt.Printf("Server delay %f seconds \n", numSeconds)
		time.Sleep(time.Duration(numSeconds) * time.Second)

		msg := <- s.channel
		fmt.Printf("Server: Received <%s> from client \n", msg)

		for _, channel := range s.clientChannels {

			channel <- msg

		}
	}
}

func (s *server) addClientChannel(c client) []chan message{
	fmt.Printf("Registering %s \n", c.name)
	s.clientChannels = append(s.clientChannels, c.clientChannel)
	return s.clientChannels
}

type client struct {
	name string
	clientChannel chan message
	server server
}



// create a channel
func NewClient(name string, s server) *client {

	clientChannel := make(chan message)
	c := client{name, clientChannel, s}
	return &c

}

func (c client) registerWithServer(s server){
	c.server = s
	s.clientChannels = s.addClientChannel(c)
}

func (c client) ping(){
	var clientMessage message;
	for {
		time.Sleep(2 * time.Second)
		clientMessage = message{c.name, fmt.Sprintf("Hello, I am %s", c.name)}
		c.server.channel <- clientMessage
	}

}

func (c client) listen(){
	for {
		broadCastedMessage:= <- c.clientChannel
		fmt.Printf("Client %s: Received '%s' from server \n", c.name, broadCastedMessage)
	}

}



