package main

import (
	"flag"
	"log"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var diskName string

type Node struct {
	inode uint64
	name  string
}

var inode uint64
var Usage = func() {
	log.Printf("Usage of %s:\n", os.Args[0])
	log.Printf("  %s MOUNTPOINT  diskName\n", os.Args[0])
	flag.PrintDefaults()
}

func NewInode() uint64 {
	inode += 1
	return inode
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	if flag.NArg() != 2 {
		Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	diskName = flag.Arg(1)

	c, err := fuse.Mount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	if p := c.Protocol(); !p.HasInvalidate() {
		log.Panicln("kernel FUSE support is too old to have invalidations: version %v", p)
	}
	srv := fs.New(c, nil)
	fd, _ := syscall.Open(diskName, os.O_RDWR, 0777)

	files := startFS(fd)
	syscall.Close(fd)
	var filesMeta []*File
	copy(filesMeta, files)
	for _, file := range files {
		file.inode = NewInode()
	}
	for _, file := range filesMeta {
		file.inode = NewInode()
		file.name = file.name + "_1001.meta"
	}
	files = append(files, filesMeta...)

	//filesys := &FS{
	//	&Dir{Node: Node{name: "head", inode: NewInode()}, files: &[]*File{
	//		&File{Node: Node{name: "blk_103741826_1001.meta", inode: NewInode()}, data: []byte("hello world!")},
	//		&File{Node: Node{name: "aybbg", inode: NewInode()}, data: []byte("send notes")},
	//	}, directories: &[]*Dir{
	//		&Dir{Node: Node{name: "left", inode: NewInode()}, files: &[]*File{
	//			&File{Node: Node{name: "yo", inode: NewInode()}, data: []byte("ayylmaooo")},
	//		},
	//		},
	//		&Dir{Node: Node{name: "right", inode: NewInode()}, files: &[]*File{
	//			&File{Node: Node{name: "hey", inode: NewInode()}, data: []byte("heeey, thats pretty good")},
	//		},
	//		},
	//	},
	//	}}

	filesys := &FS{
		root: &Dir{
			Node:        Node{name: "head", inode: NewInode()},
			files:       &files,
			directories: nil,
		},
	}
	log.Println("About to serve fs")
	if err := srv.Serve(filesys); err != nil {
		log.Panicln(err)
	}
	// Check if the mount process has an error to report.
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Panicln(err)
	}
}
