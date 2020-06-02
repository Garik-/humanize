package main

import (
	"log"
	"os"
)

func main() {
	f, err := os.Open("./8th_Acc_2.mid")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	decoder := newDecoder(f)
	err = decoder.decode()

	if err != nil {
		log.Println(err)
	}

	log.Printf("notes: %d\n", len(decoder.events))
}