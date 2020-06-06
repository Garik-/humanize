package midi

import (
	"bytes"
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func createCopyTmpFile(src *os.File) (*os.File, error) {
	tmp, err := ioutil.TempFile("", "test")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(tmp, src)
	if err != nil {
		os.Remove(tmp.Name())
		return nil, err
	}

	return tmp, nil
}

func writeVelocity(w io.WriteSeeker, decoder *Decoder) error {
	for _, track := range decoder.Tracks {
		for _, event := range track.Events {
			_, err := w.Seek(event.VelocityByteOffset, io.SeekStart)
			if err != nil {
				return err
			}
			err = binary.Write(w, binary.BigEndian, event.Velocity)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func equalFiles(src *os.File, dst *os.File) (bool, error) {
	_, err := src.Seek(0, io.SeekStart)
	if err != nil {
		return false, err
	}
	_, err = dst.Seek(0, io.SeekStart)
	if err != nil {
		return false, err
	}

	var a, b []byte
	a, err = ioutil.ReadAll(src)
	if err != nil {
		return false, err
	}

	b, err = ioutil.ReadAll(dst)
	if err != nil {
		return false, err
	}

	return bytes.Equal(a, b), nil
}

func TestDecoder_Decode(t *testing.T) {
	f, err := os.Open("./test.mid")
	require.NoError(t, err)

	defer f.Close()

	decoder := NewDecoder(f)
	err = decoder.Decode()
	require.NoError(t, err)

	events := decoder.Tracks[0].Events

	assert.Equal(t, 2, len(events))

	assert.Equal(t, uint8(9), events[0].MsgType)
	assert.Equal(t, uint8(72), events[0].Velocity)

	assert.Equal(t, uint8(8), events[1].MsgType)
	assert.Equal(t, uint8(64), events[1].Velocity)

	var tmp *os.File
	tmp, err = createCopyTmpFile(f)
	require.NoError(t, err)

	defer os.Remove(tmp.Name())

	err = writeVelocity(tmp, decoder)
	require.NoError(t, err)

	var isEqual bool
	isEqual, err = equalFiles(f, tmp)
	require.NoError(t, err)
	assert.Equal(t, true, isEqual)
}

func TestDecodeQuarter(t *testing.T) {
	f, err := os.Open("./test2.mid")
	require.NoError(t, err)

	defer f.Close()

	decoder := NewDecoder(f)
	err = decoder.Decode()
	require.NoError(t, err)

	t.Logf("decoder ticks in 1/4: %d", decoder.TicksPerQuarterNote)

	for _, track := range decoder.Tracks {
		for _, event := range track.Events {
			t.Logf("note: %d, type: %#x, delta: %d", event.Note, event.MsgType, event.TimeDelta)
		}
	}
}

func TestRange(t *testing.T) {
	var n int64 = 480
	r := newMyRange(0, n)

	var x int64 = 90
	for {
		if r.contains(x) == true {
			break
		}
		t.Log("step")

		if x > n {
			r.stepBy(int(x / n))
		} else {
			r.stepBy(1)
		}
	}

	t.Logf("cnt: %d, min - %d, max - %d, position: %d", r.cnt, r.lowerBound, r.upperBound, r.position())
}
