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

	cases := []int{2, 1, 3}
	i := 0

	for _, track := range decoder.Tracks {
		for _, event := range track.Events {
			if event.MsgType == 0x09 {
				assert.Equal(t, cases[i], event.QuarterPosition)
				i++
			}
		}
	}
}
