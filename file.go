package main

import (
	//"fmt"
	"log"
	"os"

	//"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"bazil.org/fuse/fuseutil"
	"context"
)

type File struct {
	Node
	//data []byte
	length uint64
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Println("Requested Attr for File", f.name, "has data size", f.length)
	a.Inode = f.inode
	a.Mode = 0777
	a.Size = f.length
	return nil
}
func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	log.Println("Requested Read on File", f.name)
	name := f.name
	bBlockId, flag := match(name)

	fd, err := syscall.Open("/dev/sdb", os.O_RDWR, 0777)
	if err != nil {
		log.Fatal(err)
	}
	output := readFile(fd, flag, bBlockId)
	fuseutil.HandleRead(req, resp, output)
	err = syscall.Close(fd)
	if err != nil {
		return err
	}
	return nil
}

func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	log.Println("Reading all of file", f.name)
	name := f.name
	bBlockId, flag := match(name)

	fd, err := syscall.Open("/dev/sdb", os.O_RDWR, 0777)
	if err != nil {
		log.Fatal(err)
	}
	output := readFile(fd, flag, bBlockId)
	log.Println(output)
	return output, nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	log.Println("Trying to write to ", f.name, "offset", req.Offset, "dataSize:", len(req.Data), "data: ", string(req.Data))
	resp.Size = len(req.Data)

	name := f.name
	bBlockId, flag := match(name)

	fd, err := syscall.Open("/dev/sdb", os.O_RDWR, 0777)
	if err != nil {
		log.Fatal(err)
	}
	err = writeFile(fd, req.Data, flag, bBlockId)
	if err != nil {
		log.Println("写文件出错")
	}
	err = syscall.Close(fd)
	if err != nil {
		return err
	}

	//f.data = req.Data
	f.length = uint64(len(req.Data))
	log.Println("Wrote to file", f.name)
	return nil
}
func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	log.Println("Flushing file", f.name)
	return nil
}
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	log.Println("Open call on file", f.name)
	return f, nil
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	log.Println("Release requested on file", f.name)
	return nil
}

func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	log.Println("Fsync call on file", f.name)
	return nil
}
