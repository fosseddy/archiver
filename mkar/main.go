package main

import (
	"fmt"
	"os"
	"io/fs"
	"path"
	"encoding/binary"
	"log"
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
	pathnames := os.Args[1:]
	records := []record{}

	if len(pathnames) < 1 {
		fmt.Fprintln(os.Stderr, "expected file or list of files")
		os.Exit(1)
	}

	for _, path := range pathnames {
		records = append(records, createRecord(path))
	}

	// TODO: accept output file name
	out, err := os.OpenFile("out.ar", os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0644)
	fatal(err)
	defer out.Close()

	createArchive(out, records)
}

func createRecord(pathname string) record {
	fi, err := os.Stat(pathname)
	fatal(err)

	if fi.IsDir() {
		return createDirKind(pathname, fi)
	}

	return createFileKind(pathname, fi)
}

func createDirKind(pathname string, fi fs.FileInfo) record {
	r := record{}

	r.kind = kindDir
	r.name = fi.Name()
	r.perm = fi.Mode().Perm()
	r.size = 0

	ds, err := os.ReadDir(pathname)
	fatal(err)

	for _, d := range ds {
		child := createRecord(path.Join(pathname, d.Name()))
		r.size += 1
		r.children = append(r.children, child)
	}

	return r
}

func createFileKind(pathname string, fi fs.FileInfo) record {
	r := record{}

	r.kind = kindFile
	r.name = fi.Name()
	r.size = fi.Size()
	r.perm = fi.Mode().Perm()

	bytes, err := os.ReadFile(pathname)
	fatal(err)
	r.content = bytes

	return r
}

func createArchive(out *os.File, records []record) {
	for _, r := range records {
		out.WriteString(r.name)
		out.Write([]byte{0})

		binary.Write(out, binary.LittleEndian, r.size)
		binary.Write(out, binary.LittleEndian, r.kind)
		binary.Write(out, binary.LittleEndian, r.perm)

		if r.kind == kindDir {
			createArchive(out, r.children)
		} else {
			out.Write(r.content)
		}
	}
}

func fatal(e error) {
	if e != nil {
		log.Fatalf("%+v\n", e)
	}
}
