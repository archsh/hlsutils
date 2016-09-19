/****
 * another test.
 *
 */

package main

import (
	"fmt"
	"time"
)

func ready(w string, s int, c *chan int) {
	time.Sleep(time.Duration(s) * time.Second)
	fmt.Printf("%s is ready!\n", w)
	*c <- 1
}

func main() {
	c := make(chan int)
	go ready("Coffee", 5, &c)
	go ready("Tea", 2, &c)
	fmt.Println("I'm waiting...")
	<-c
	<-c
}
