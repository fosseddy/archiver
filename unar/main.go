package main

import (
	"fmt"
	"os"
	"io"
	"io/fs"
	"encoding/binary"
	"bytes"
	"path"
	"log"
	"errors"
)

type recordKind int8

const (
	kindFile recordKind = iota
	kindDir
)

type record struct {
	name string
	size int64
	kind recordKind
	perm fs.FileMode
	content []byte
	children []record
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "expected archive")
		os.Exit(1)
	}

	archive := os.Args[1]

	b, err := os.ReadFile(archive)
	fatal(err)

	data := bytes.NewReader(b)
	records := []record{}

	for {
		r, err := readRecord(data)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fatal(err)
		}
		records = append(records, r)
	}

	createRecords(records, path.Dir(archive))
}

func readRecord(data *bytes.Reader) (record, error) {
	r := record{}

	name := []byte{}
	for {
		ch, err := data.ReadByte()
		if err != nil {
			return r, err
		}
		if ch == 0 {
			break
		}
		name = append(name, ch)
	}
	r.name = string(name)

	err := binary.Read(data, binary.LittleEndian, &r.size)
	err = binary.Read(data, binary.LittleEndian, &r.kind)
	err = binary.Read(data, binary.LittleEndian, &r.perm)
	if err != nil {
		return r, err
	}

	if r.size > 0 {
		if r.kind == kindDir {
			var size int64 = 0

			for {
				child, err := readRecord(data)
				if err != nil {
					return r, err
				}

				r.children = append(r.children, child)

				size += 1
				if r.size == size {
					break
				}
			}
		} else {
			content := make([]byte, r.size)
			_, err := data.Read(content)
			if err != nil {
				return r, err
			}
			r.content = content
		}
	}

	return r, nil
}

func createRecords(records []record, dirpath string) {
	for _, r := range records {
		pathname := path.Join(dirpath, r.name)

		if r.kind == kindFile {
			fd, err := os.OpenFile(pathname, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, r.perm)
			fatal(err)
			_, err = fd.Write(r.content)
			fatal(err)
			fd.Close()
		} else {
			err := os.Mkdir(pathname, r.perm)
			fatal(err)
			createRecords(r.children, pathname)
		}
	}
}

func fatal(e error) {
	if e != nil {
		log.Fatalf("%+v\n", e)
	}
}
