package midi

import (
	"encoding/binary"
	"io"
)

// add offset
func (d *Decoder) readByte() (byte, error) {
	var b byte
	err := binary.Read(d.r, binary.BigEndian, &b)
	if err == nil {
		d.offset += 1 // read byte
	}
	return b, err
}

func (d *Decoder) uint7() (uint8, error) {
	b, err := d.readByte()
	if err != nil {
		return 0, err
	}
	return b & 0x7f, nil
}

// VarLen returns the variable length value at the exact parser location.
func (d *Decoder) varLen() (val uint32, err error) {
	buf := []byte{}
	var lastByte bool

	for !lastByte {
		b, err := d.readByte()
		if err != nil {
			return 0, err
		}
		buf = append(buf, b)
		lastByte = b>>7 == 0x0
	}

	val, _ = decodeVarint(buf)
	return val, nil
}

func (d *Decoder) varLenTxt() error {
	l, err := d.varLen()
	if err != nil {
		return err
	}
	d.offset += int64(l)
	_, err = d.r.Seek(d.offset, io.SeekStart)
	return err
}

func (d *Decoder) IDnSize() ([4]byte, error) {
	var ID [4]byte
	if err := binary.Read(d.r, binary.BigEndian, &ID); err != nil {
		return ID, err
	}
	d.offset += 4 // [4]byte ID

	if _, err := d.r.Seek(4, io.SeekCurrent); err != nil {
		return ID, err
	}
	d.offset += 4 // uint32 blockSize

	return ID, nil
}

func quarterPosition(absTicks int64, ticksPerQuarterNote int64) int {
	r := newMyRange(0, ticksPerQuarterNote)

	for {
		if r.contains(absTicks) == true {
			break
		}

		if absTicks > ticksPerQuarterNote {
			r.stepBy(int(absTicks / ticksPerQuarterNote))
		} else {
			r.stepBy(1)
		}
	}

	return r.position()
}
