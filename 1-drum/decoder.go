package drum

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
)

// DecodeFile decodes the drum machine file found at the provided path
// and returns a pointer to a parsed pattern which is the entry point to the
// rest of the data.
func DecodeFile(path string) (*Pattern, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := decoder{Reader: &errorReader{Reader: file}}
	p := d.decode()

	err = d.Reader.Err
	if err != nil {
		err = d.Err
	}
	return p, err
}

type errorReader struct {
	Reader io.Reader
	Err    error
}

func (r *errorReader) Read(b []byte) int {
	var num int
	if r.Err != nil {
		return num
	} else {
		n, err := r.Reader.Read(b)
		if err != nil {
			r.Err = err
		} else {
			num = n
		}
	}
	return num
}

func (r *errorReader) BinaryRead(order binary.ByteOrder, data interface{}) {
	if r.Err != nil {
		return
	}

	binary.Read(r.Reader, order, data)
}

type decoder struct {
	Reader *errorReader
	Err    error //the first error encountered by the decoder
}

func (d *decoder) decode() *Pattern {
	length := d.getFileLength()

	//Now that we know the length
	//prevent reading passed the limit of the file
	d.Reader.Reader = &io.LimitedReader{R: d.Reader.Reader, N: int64(length)}

	p := &Pattern{}
	p.Header = d.decodeHeader()

	//now decode the Tracks
	p.Tracks = make([]*track, 0)
	t := d.decodeNexttrack()
	for t != nil {
		p.Tracks = append(p.Tracks, t)
		t = d.decodeNexttrack()
	}
	return p
}

//file length is the remaining expected number of bytes
//This include the remaining Header (36 bytes) plus the
//encoded data
func (d *decoder) getFileLength() int {
	p := make([]byte, 14)
	n := d.Reader.Read(p)

	if n >= 0 && (n < 14 || string(p[:6]) != "SPLICE") {
		d.Err = errors.New("'SPLICE' identifier not found")
		return -1
	}

	length := int(p[13])
	if length < 36 {
		d.Err = errors.New("Malformed SPLICE file, Header is not 50 bytes")
		return -1
	}

	return length
}

//Assumes the next 36 bytes in r are the Header bytes
//The first 32 are ASCII characters representing the HW Version
//The last 4 bytes correspond to a float representing the tempo
func (d *decoder) decodeHeader() *header {
	if d.Err != nil {
		return nil
	}

	h := &header{}

	versionBytes := make([]byte, 32)
	d.Reader.Read(versionBytes)
	h.Version = strings.Trim(string(versionBytes), string(byte(0)))

	d.Reader.BinaryRead(binary.LittleEndian, &h.Tempo)

	return h
}

func (d *decoder) decodeNexttrack() *track {
	if d.Err != nil {
		return nil
	}

	t := &track{}

	//First byte is the id of the track
	idByte := make([]byte, 1)
	d.Reader.Read(idByte)
	t.ID = int(idByte[0])

	//Then there are 3 blank bytes
	throwAway := make([]byte, 3)
	d.Reader.Read(throwAway)

	//The next byte is the legnth of the track's name
	nameLength := make([]byte, 1)
	d.Reader.Read(nameLength)

	//Now parse the track's name
	nameRaw := make([]byte, int(nameLength[0]))
	d.Reader.Read(nameRaw)
	t.Name = string(nameRaw)

	//Finally, the next 16 bytes represent the measure
	rawNotes := make([]byte, 16)
	d.Reader.Read(rawNotes)

	for i, note := range rawNotes {
		if note != 0 {
			t.Bars[i] = true
		} else {
			t.Bars[i] = false
		}
	}

	if d.Reader.Err != nil {
		return nil
	} else {
		return t
	}
}

//Pattern is a High level representation of the drum pattern
//Header contains information about version and temp
//track holds information about the individual track:
//name, id, bars
type Pattern struct {
	Header *header
	Tracks []*track
}

type header struct {
	Version string
	Tempo   float32
}

type track struct {
	Name string
	ID   int
	Bars [16]bool
}
