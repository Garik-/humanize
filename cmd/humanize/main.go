package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Garik-/humanize/pkg/midi"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

var (
	databaseFlag = flag.String("d", "", "The path to the database json file")
	inFlag = flag.String("i", "", "Input midi file")
	outFlag = flag.String("o", "", "Output midi file")
	minFlag = flag.Int("min", 0, "Min velocity")
	maxFlag = flag.Int("max", 127, "Max velocity")
)

type velocityMap map[uint8]map[uint8][]int

func importDatabase(name string) (velocityMap, error) {
	jsonFile, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	var bytes []byte
	bytes, err = ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var data velocityMap
	err = json.Unmarshal(bytes, &data)
	return data, err
}

func randVelocity(velocities []int, min int, max int) uint8 {
	for {
		rand.Seed(time.Now().UTC().UnixNano())
		velocity := velocities[rand.Intn(len(velocities))]
		if velocity > min && velocity < max {
			return uint8(velocity)
		}
	}
}

func writeRandVelocity(w io.WriteSeeker, decoder *midi.Decoder, data velocityMap) error {
	for _, event := range decoder.Events {
		if msgType, ok := data[event.Note]; ok {
			if velocities, ok := msgType[event.MsgType]; ok {
				velocity := randVelocity(velocities, *minFlag, *maxFlag)
				_, err := w.Seek(event.VelocityByteOffset, io.SeekStart)
				if err != nil {
					return err
				}

				err = binary.Write(w, binary.BigEndian, velocity)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s \n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *databaseFlag == "" {
		flag.Usage()
		return
	}

	data, err := importDatabase(*databaseFlag)
	if err != nil {
		log.Fatal(err)
	}

	var in, out *os.File
	in, err = os.Open(*inFlag)
	if err != nil {
		log.Fatal(err)
	}

	out, err = os.OpenFile(*outFlag, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		in.Close()
		log.Fatal(err)
	}

	defer func() {
		out.Close()
		in.Close()
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		log.Fatal(err)
	}

	_, err = in.Seek(0, 0)
	if err != nil {
		log.Fatal(err)
	}
	_, err = out.Seek(0, 0)
	if err != nil {
		log.Fatal(err)
	}

	decoder := midi.NewDecoder(in)
	err = decoder.Decode()

	if err != nil {
		log.Fatal(err)
	}

	err = writeRandVelocity(out, decoder, data)

	if err != nil {
		log.Fatal(err)
	}
}