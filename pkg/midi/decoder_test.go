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
	for _, event := range decoder.Events {
		_, err := w.Seek(event.VelocityByteOffset, io.SeekStart)
		if err != nil {
			return err
		}
		err = binary.Write(w, binary.BigEndian, event.Velocity)
		if err != nil {
			return err
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

	assert.Equal(t, 2, len(decoder.Events))

	assert.Equal(t, uint8(9), decoder.Events[0].MsgType)
	assert.Equal(t, uint8(72), decoder.Events[0].Velocity)

	assert.Equal(t, uint8(8), decoder.Events[1].MsgType)
	assert.Equal(t, uint8(64), decoder.Events[1].Velocity)

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
