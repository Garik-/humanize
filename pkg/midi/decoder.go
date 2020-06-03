package midi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type nextChunkType int

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

type Event struct {
	msgType            uint8
	note               uint8
	velocity           uint8
	velocityByteOffset int64
}

type Decoder struct {
	r         io.ReadSeeker
	Events    []*Event
	lastEvent *Event
	offset    int64
}

func (d *Decoder) Decode() error {
	if _, err := d.r.Seek(0, io.SeekStart); err != nil {
		return err
	}

	var code [4]byte
	d.offset = 0

	if err := binary.Read(d.r, binary.BigEndian, &code); err != nil {
		return err
	}

	if code != headerChunkID {
		return fmt.Errorf("%s - %v", ErrFmtNotSupported, code)
	}

	d.offset += 4 // [4]byte code

	var headerSize uint32
	if err := binary.Read(d.r, binary.BigEndian, &headerSize); err != nil {
		return err
	}

	if headerSize != 6 {
		return fmt.Errorf("%s - expected header size to be 6, was %d", ErrFmtNotSupported, headerSize)
	}

	d.offset += 4 + 2 + 2 + 2 // uint32 headerSize + uint16 Format + uint16 NumTracks + uint16 Division

	if _, err := d.r.Seek(d.offset, io.SeekStart); err != nil {
		return err
	}

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

	_, err = d.r.Seek(0, io.SeekStart)
	return err
}

func (d *Decoder) parseTrack() (nextChunkType, error) {
	id, err := d.IDnSize()
	if err != nil {
		return trackChunk, err
	}
	if id != trackChunkID {
		return trackChunk, fmt.Errorf("%s - expected track chunk ID %v, got %v", ErrUnexpectedData, trackChunkID, id)
	}

	return eventChunk, nil
}

func (d *Decoder) parseEvent() (nextChunkType, error) {
	var err error

	_, err = d.varLen()
	if err != nil {
		return eventChunk, err
	}

	// status byte give us the msg type and channel.
	statusByte, err := d.readByte()
	if err != nil {
		return eventChunk, err
	}

	e := new(Event)
	e.msgType = (statusByte & 0xF0) >> 4

	if statusByte&0x80 == 0 {
		if d.lastEvent != nil && isVoiceMsgType(d.lastEvent.msgType) {
			e.msgType = d.lastEvent.msgType

			d.offset -= 1
			if _, err := d.r.Seek(-1, io.SeekCurrent); err != nil {
				return eventChunk, err
			}
		}
	}

	if e.msgType == 0 {
		return eventChunk, nil
	}

	d.lastEvent = e

	nextChunk := eventChunk

	// Extract values based on message type
	switch e.msgType {

	case 0x2, 0x3, 0x4, 0x5, 0x6, 0xC, 0xD:
		if _, err := d.r.Seek(1, io.SeekCurrent); err != nil {
			return eventChunk, err
		}
		d.offset += 1

	case 0xB, 0xE:
		if _, err := d.r.Seek(2, io.SeekCurrent); err != nil {
			return eventChunk, err
		}
		d.offset += 2

	case 0x8:
		if e.note, err = d.uint7(); err != nil {
			return eventChunk, err
		}
		e.velocityByteOffset = d.offset
		if e.velocity, err = d.uint7(); err != nil {
			return eventChunk, err
		}
		d.Events = append(d.Events, e)

	case 0x9:
		if e.note, err = d.uint7(); err != nil {
			return eventChunk, err
		}
		e.velocityByteOffset = d.offset
		if e.velocity, err = d.uint7(); err != nil {
			return eventChunk, err
		}
		d.Events = append(d.Events, e)

	case 0xA:
		if e.note, err = d.uint7(); err != nil {
			return eventChunk, err
		}
		e.velocityByteOffset = d.offset
		if e.velocity, err = d.uint7(); err != nil {
			return eventChunk, err
		}
		d.Events = append(d.Events, e)
	case 0xF:
		var ok bool
		nextChunk, ok, err = d.parseMetaMsg()
		// early exit without adding the event to the track
		if err != nil || !ok {
			return nextChunk, err
		}

	default:
		return eventChunk, nil
	}

	return nextChunk, err
}

func (p *Decoder) parseMetaMsg() (nextChunkType, bool, error) {
	if _, err := p.readByte(); err != nil {
		return eventChunk, false, err
	}

	err := p.varLenTxt()
	if err != nil {
		return eventChunk, false, err
	}
	return eventChunk, true, nil
}

func NewDecoder(r io.ReadSeeker) *Decoder {
	return &Decoder{r: r, offset: 0}
}