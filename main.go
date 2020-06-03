package main

import (
	"encoding/binary"
	"io"
	"log"
	"os"
)

func testWrite(src *os.File, d *decoder) error {
	f, err := os.Create("./trololo.mid")
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(f, src)
	if err != nil {
		return err
	}

	var w io.WriteSeeker = f
	for _, event := range d.events {
		w.Seek(event.velocityByteOffset, io.SeekStart)
		if err := binary.Write(w, binary.BigEndian, event.velocity); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	f, err := os.Open("./main.mid")
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

	err = testWrite(f, decoder)
	if err != nil {
		log.Println(err)
	}
}
