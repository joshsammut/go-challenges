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

	p, err := decode(file)

	return p, nil
}

type decoder struct {
	r io.Reader
}

func decode(r io.Reader) (*Pattern, error) {
	length, err := getFileLength(r)
	if err != nil {
		return nil, errors.New("Malformed SPLICE file, " + err.Error())
	}

	if length < 36 {
		return nil, errors.New("Malformed SPLICE file, Header is not 50 bytes")
	}

	p := &Pattern{}
	p.Header, err = decodeHeader(r)
	if err != nil {
		return nil, errors.New("Error decoding Header: " + err.Error())
	}

	//now decode the Tracks
	p.Tracks = make([]*track, 0)
	for i := 0; i < (length - 36); {
		nexttrack, trackLength, err := decodeNexttrack(r)
		if err != nil {
			return nil, errors.New("Error decoding track" + err.Error())
		}
		p.Tracks = append(p.Tracks, nexttrack)
		i += trackLength
	}
	return p, err
}

//file length is the remaining expected number of bytes
//This include the remaining Header (36 bytes) plus the
//encoded data
func getFileLength(r io.Reader) (int, error) {
	p := make([]byte, 14)
	n, err := r.Read(p)

	if n >= 0 && (n < 14 || string(p[:6]) != "SPLICE") {
		err = errors.New("'SPLICE' identifier not found")
	}

	if err != nil {
		return -1, err
	}

	return int(p[13]), nil
}

//Assumes the next 36 bytes in r are the Header bytes
//The first 32 are ASCII characters representing the HW Version
//The last 4 bytes correspond to a float representing the tempo
func decodeHeader(r io.Reader) (*header, error) {
	h := &header{}

	versionBytes := make([]byte, 32)
	_, err := r.Read(versionBytes)
	if err != nil {
		return nil, err
	}
	h.Version = strings.Trim(string(versionBytes), string(byte(0)))

	err = binary.Read(r, binary.LittleEndian, &h.Tempo)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func decodeNexttrack(r io.Reader) (*track, int, error) {
	t := &track{}

	//First byte is the id of the track
	var idByte byte
	err := binary.Read(r, binary.LittleEndian, &idByte)
	if err != nil {
		return nil, -1, err
	}
	t.ID = int(idByte)

	//Then there are 3 blank bytes
	var throwAway [3]byte
	err = binary.Read(r, binary.LittleEndian, &throwAway)
	if err != nil {
		return nil, -1, err
	}

	//The next byte is the legnth of the track's name
	var nameLength byte
	err = binary.Read(r, binary.BigEndian, &nameLength)

	//Now parse the track's name
	nameRaw := make([]byte, nameLength)
	_, err = r.Read(nameRaw)
	if err != nil {
		return nil, -1, err
	}
	t.Name = string(nameRaw)

	//Finally, the next 16 bytes represent the measure
	err = binary.Read(r, binary.BigEndian, &t.Bars)
	if err != nil {
		return nil, -1, err
	}

	return t, (1 + 3 + 1 + len(t.Name) + 16), nil
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
	Bars [16]byte
}
