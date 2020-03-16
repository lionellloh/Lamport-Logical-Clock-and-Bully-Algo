package main

import (
	"fmt"
	"time"
)

const SLEEP_DURATION = 5

func main() {
	//var s string
	var numinstances int = 5 // at least 3 required for the program to work
	var instances []*Instance
	var killChannels []chan bool

	fmt.Println("=================================================")
	fmt.Println("                  INITIALIZATION                 ")
	fmt.Println("=================================================")

	fmt.Printf("[INITIALIZATION] %d instances will be set up. Election messages will be passed around", numinstances)
	time.Sleep(time.Duration(SLEEP_DURATION * time.Second))

	// create and start all instances
	for i := 0; i < numinstances; i++ {
		node := ConstructNewNode(i)
		ch := make(chan bool)
		go node.start(ch, instances)
		instances = append(instances, node)
		killChannels = append(killChannels, ch)
	}

	time.Sleep(time.Duration(SLEEP_DURATION * time.Second))
	fmt.Println("=================================================")
	fmt.Println("                    WORST CASE                    ")
	fmt.Println("=================================================")
	fmt.Println("\n[WORST CASE] Simulate current coordinator going down, discovery by lowest node.")
	time.Sleep(time.Duration(SLEEP_DURATION * time.Second))

	// kill coordinator, make node 0 contact it
	killChannels[numinstances-1] <- true
	fmt.Println("Coordinator node", instances[numinstances-1].ID, "has been killed")
	instances[0].SendMessage(instances[numinstances-1], Message{MiscMsg, instances[0]})

	time.Sleep(time.Duration(int64(2*(numinstances-1)+1)) * time.Second)
	fmt.Println("\nNext, the previous coordinator will be revived.")

	// revive highest ID node
	go instances[numinstances-1].start(killChannels[numinstances-1], instances[:numinstances-1])

	time.Sleep(time.Duration(SLEEP_DURATION * time.Second))
	fmt.Println("=================================================")
	fmt.Println("                    BEST CASE                    ")
	fmt.Println("=================================================")
	fmt.Println("\n[BEST CASE] Simulate current coordinator going down, discovery by second highest node ")
	time.Sleep(time.Duration(SLEEP_DURATION * time.Second))

	// kill coordinator, make node numinstances-2 contact it
	killChannels[numinstances-1] <- true
	fmt.Println("Coordinator node", instances[numinstances-1].ID, "has been killed")
	instances[numinstances-2].SendMessage(instances[numinstances-1], Message{MiscMsg, instances[numinstances-2]})

	time.Sleep(time.Duration(SLEEP_DURATION * time.Second))
	fmt.Println("\nTrigger election at instance 0, and the second highest ID node will be killed after election starts. Meanwhile, the highest ID node is also dead from the previous demo.")
	time.Sleep(time.Duration(SLEEP_DURATION * time.Second))

	instances[0].startElection()
	killChannels[numinstances-2] <- true
	fmt.Println("Instance", numinstances-2, "has been killed")

	time.Sleep(time.Duration(int64(2*(numinstances-1)+1)) * time.Second)
	time.Sleep(time.Duration(SLEEP_DURATION * time.Second))
}
