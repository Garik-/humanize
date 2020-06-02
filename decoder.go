package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type nextChunkType int

var offset int64

const (
	eventChunk nextChunkType = iota + 1
	trackChunk
)

var (
	headerChunkID = [4]byte{0x4D, 0x54, 0x68, 0x64}
	trackChunkID  = [4]byte{0x4D, 0x54, 0x72, 0x6B}

	// ErrFmtNotSupported is a generic error reporting an unknown format.
	ErrFmtNotSupported = errors.New("format not supported")
	// ErrUnexpectedData is a generic error reporting that the parser encountered unexpected data.
	ErrUnexpectedData = errors.New("unexpected data content")
)

type event struct {
	msgType            uint8
	note               uint8
	velocity           uint8
	velocityByteOffset int64
}

type decoder struct {
	r         io.ReadSeeker
	events    []*event
	lastEvent *event
}

func (d *decoder) decode() error {
	var code [4]byte

	offset = 0

	if err := binary.Read(d.r, binary.BigEndian, &code); err != nil {
		return err
	}

	if code != headerChunkID {
		return fmt.Errorf("%s - %v", ErrFmtNotSupported, code)
	}

	offset += 4 // [4]byte code

	var headerSize uint32
	if err := binary.Read(d.r, binary.BigEndian, &headerSize); err != nil {
		return err
	}

	if headerSize != 6 {
		return fmt.Errorf("%s - expected header size to be 6, was %d", ErrFmtNotSupported, headerSize)
	}

	offset += 4 // uint32 headerSize
	offset += 2 // uint16 Format
	offset += 2 // uint16 NumTracks
	offset += 2 // uint16 Division

	d.r.Seek(offset, io.SeekStart)

	nextChunk, err := d.parseTrack()
	if err != nil {
		return err
	}

	for err != io.EOF {
		switch nextChunk {
		case eventChunk:
			nextChunk, err = d.parseEvent()
		case trackChunk:
			nextChunk, err = d.parseTrack()
		}

		if err != nil && err != io.EOF {
			return err
		}
	}

	return nil
}

func (d *decoder) parseTrack() (nextChunkType, error) {
	id, err := d.IDnSize()
	if err != nil {
		return trackChunk, err
	}
	if id != trackChunkID {
		return trackChunk, fmt.Errorf("%s - expected track chunk ID %v, got %v", ErrUnexpectedData, trackChunkID, id)
	}

	return eventChunk, nil
}

// add offset
func (d *decoder) readByte() (byte, error) {
	var b byte
	err := binary.Read(d.r, binary.BigEndian, &b)
	if err == nil {
		offset += 1 // read byte
	}
	return b, err
}

func (d *decoder) uint7() (uint8, error) {
	b, err := d.readByte()
	if err != nil {
		return 0, err
	}
	return (b & 0x7f), nil
}

// VarLen returns the variable length value at the exact parser location.
func (d *decoder) varLen() (val uint32, err error) {
	buf := []byte{}
	var lastByte bool

	for !lastByte {
		b, err := d.readByte()
		if err != nil {
			return 0, err
		}
		buf = append(buf, b)
		lastByte = (b>>7 == 0x0)
	}

	val, _ = decodeVarint(buf)
	return val, nil
}

func (p *decoder) parseEvent() (nextChunkType, error) {
	var err error

	_, err = p.varLen()
	if err != nil {
		return eventChunk, err
	}

	// status byte give us the msg type and channel.
	statusByte, err := p.readByte()
	if err != nil {
		return eventChunk, err
	}


	e := new(event)
	e.msgType = (statusByte & 0xF0) >> 4

	if statusByte&0x80 == 0 {
		if p.lastEvent != nil && isVoiceMsgType(p.lastEvent.msgType) {
			e.msgType = p.lastEvent.msgType

			offset -= 1
			p.r.Seek(offset, io.SeekStart)
		}
	}

	if e.msgType == 0 {
		return eventChunk, nil
	}

	p.lastEvent = e

	nextChunk := eventChunk

	// Extract values based on message type
	switch e.msgType {

	case 0x2, 0x3, 0x4, 0x5, 0x6, 0xC, 0xD:
		offset += 1
		p.r.Seek(offset, io.SeekStart)

	case 0xB, 0xE:
		offset += 2
		p.r.Seek(offset, io.SeekStart)

	case 0x8:
		if e.note, err = p.uint7(); err != nil {
			return eventChunk, err
		}
		if e.velocity, err = p.uint7(); err != nil {
			return eventChunk, err
		}

		e.velocityByteOffset = offset

		p.events = append(p.events, e)

	case 0x9:
		if e.note, err = p.uint7(); err != nil {
			return eventChunk, err
		}
		if e.velocity, err = p.uint7(); err != nil {
			return eventChunk, err
		}

		e.velocityByteOffset = offset

		p.events = append(p.events, e)

	case 0xA:
		if e.note, err = p.uint7(); err != nil {
			return eventChunk, err
		}
		// aftertouch value
		if e.velocity, err = p.uint7(); err != nil {
			return eventChunk, err
		}

		e.velocityByteOffset = offset

		p.events = append(p.events, e)

	case 0xF:
		var ok bool
		nextChunk, ok, err = p.parseMetaMsg(e)
		// early exit without adding the event to the track
		if err != nil || !ok {
			return nextChunk, err
		}

	default:
		return eventChunk, nil
	}

	return nextChunk, err
}

func (d *decoder) varLenTxt() error {
	l, err := d.varLen()
	if err != nil {
		return err
	}
	offset += int64(l)
	d.r.Seek(offset, io.SeekStart)

	return nil
}

func (p *decoder) parseMetaMsg(e *event) (nextChunkType, bool, error) {
	if _, err := p.readByte(); err != nil {
		return eventChunk, false, err
	}

	err := p.varLenTxt()
	if err != nil {
		return eventChunk, false, err
	}
	return eventChunk, true, nil
}

func (d *decoder) IDnSize() ([4]byte, error) {
	var ID [4]byte
	if err := binary.Read(d.r, binary.BigEndian, &ID); err != nil {
		return ID, err
	}
	offset += 4 // [4]byte ID

	d.r.Seek(4, io.SeekCurrent) // uint32 blockSize
	offset += 4

	return ID, nil
}

func newDecoder(r io.ReadSeeker) *decoder {
	return &decoder{r: r}
}

func decodeVarint(buf []byte) (x uint32, n int) {
	if len(buf) < 1 {
		return 0, 0
	}

	if buf[0] <= 0x80 {
		return uint32(buf[0]), 1
	}

	var b byte
	for _, b = range buf {
		x = x << 7
		x |= uint32(b) & 0x7F
		n++
		if b&0x80 == 0 {
			return x, n
		}
	}

	return x, n
}

func isVoiceMsgType(b byte) bool {
	return 0x8 <= b && b <= 0xE
}
