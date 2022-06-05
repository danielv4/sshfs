/*
 * sshfs.go
 *
 * Copyright 2022 Daniel Vanderloo
 */
/*
 * This file is part of Cgofuse.
 *
 * It is licensed under the MIT license. The full license text can be found
 * in the License.txt file at the root of this project.
 */

package main

import (
	"os"
	"fmt"
	
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"	
	
	"github.com/winfsp/cgofuse/fuse"
	
	"io"
	"path"
)



// Cache
type Node struct {
	
	Path  string
	IsDir bool
	Size  int
	fp *sftp.File
	mknod *sftp.File
}


type Sshfs struct {
	fuse.FileSystemBase
	client *sftp.Client
	nodes map[string]*Node
}




func (self *Sshfs) Open(path string, flags int) (errc int, fh uint64) {

	fmt.Printf("Open() %s\n", path)

	if _, found := self.nodes[path]; found {
	
		//OpenFile(path string, f int) (*File, error)
	
		fp, err := self.client.Open(path)
		if err != nil {
			fmt.Println(err)
			return -fuse.ENOENT, ^uint64(0)
		}
		self.nodes[path].fp = fp

		return 0, 0
		
	} else {
		return -fuse.ENOENT, ^uint64(0)
	}
}


func (self *Sshfs) Opendir(path string) (errc int, fh uint64) {
	fmt.Printf("Opendir() %s\n", path)
	return 0, 0
}


func (self *Sshfs) Unlink(path string) (errc int) {
	
	err := self.client.Remove(path)
	if err != nil {
		fmt.Println(err)
	}
	return 0
}


func (self *Sshfs) Rmdir(path string) (errc int) {
	
	err := self.client.RemoveDirectory(path)
	if err != nil {
		fmt.Println(err)
	}
	return 0
}


func (self *Sshfs) Rename(oldpath string, newpath string) (errc int) {

	fmt.Printf("Rename() %s %s\n", oldpath, newpath)

	err := self.client.Rename(oldpath, newpath)
	if err != nil {
		fmt.Println(err)
	}	
	
	info, err := self.client.Stat(newpath)	
	if err != nil {
		fmt.Println(err)
	}	
	
	node := new(Node)
	node.IsDir = info.IsDir()
	node.Size = int(info.Size())
	node.Path = newpath	
	self.nodes[newpath] = node
	
	return 0
}


func (self *Sshfs) Mkdir(path string, mode uint32) (errc int) {
	// pre_write
	// create file 
	// then open
	fmt.Printf("Mkdir => %s\n", path)
	
	err := self.client.MkdirAll(path)
	if err != nil {
		return
	}
	
	node := new(Node)
	node.IsDir = true
	node.Size = 0
	node.Path = path	
	self.nodes[path] = node

	return
}


func (self *Sshfs) Mknod(path string, mode uint32, dev uint64) (errc int) {

	// pre_write
	// create file 
	// then open
	fmt.Printf("Mknod => %s\n", path)
	

	fp, err := self.client.Create(path)
	if err != nil {
		return
	}
	
	node := new(Node)
	node.IsDir = false
	node.Size = 0
	node.Path = path	
	node.mknod = fp
	self.nodes[path] = node

	return
}


func (self *Sshfs) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {

	fmt.Printf("Getattr() %s\n", path)
	//fmt.Printf("%+v\n", self.nodes)
	
	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 0777
		return 0	
	} else if node, found := self.nodes[path]; found {
	
		if node.IsDir == true {
			stat.Mode = fuse.S_IFDIR | 0777
		} else {
			stat.Mode = fuse.S_IFREG | 0777
			stat.Size = int64(node.Size)	
		}

		return 0		
	} else {
		return -fuse.ENOENT
	}
}


func (self *Sshfs) Write(path string, buff []byte, ofst int64, fh uint64) (n int) {

	fmt.Printf("Write() %s\n", path)
	fmt.Printf("Write(?) %d\n", len(buff))

	if node, found := self.nodes[path]; found {
	
		n, err := node.mknod.WriteAt(buff, ofst)
		if nil != err && io.EOF != err {
			//n = fuseErrc(err)
			return 0
		}
		self.nodes[path].Size += n

		return n		
	} else {
		n = -fuse.EIO
		return 0 
	}

	return 0
}


func (self *Sshfs) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {

	fmt.Printf("Read() %s\n", path)

	if node, found := self.nodes[path]; found {
	
		n, err := node.fp.ReadAt(buff, ofst)
		if nil != err && io.EOF != err {
			//n = fuseErrc(err)
			return 0
		}

		return n		
	} else {
		n = -fuse.EIO
		return 0 
	}

	return 0
}


func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}


func (self *Sshfs) updateInodes(fpath string, entries []os.FileInfo) {

	var mFiles []string
	
	for _, entry := range entries {
		
		vpath := ""
		if fpath == "/" {
			vpath = fpath + entry.Name()
		} else {
			vpath = fpath + "/" + entry.Name()
		}
		
		mFiles = append(mFiles, vpath)
	}	
	
	
	for k, v := range self.nodes { 
	
		dir := path.Dir(k)
		if fpath == dir {
			
			// k
			if stringInSlice(k, mFiles) == false {
				fmt.Printf("[outdated inode] key[%s] value[%s]\n", k, v)
			}
		}
	}
}


func (self *Sshfs) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {
	
	
	fill(".", nil, 0)
	fill("..", nil, 0)
	
	
	entries, err := self.client.ReadDir(path)
	if err != nil {
		fmt.Println(err)
	} else {
	
		//self.updateInodes(path, entries)
	
		for _, entry := range entries {
		
			fill(entry.Name(), nil, 0)
			
			// add node to Cache for Getattr()
			node := new(Node)
			node.IsDir = entry.IsDir()
			node.Size = int(entry.Size())
			if path == "/" {
				node.Path = path + entry.Name()
			} else {
				node.Path = path + "/" + entry.Name()
			}
			
			
			
			fmt.Printf("%+v\n", node)
			
			self.nodes[node.Path] = node
			
			
		}
	}	
	
	
	// update Cache when file deleted by another session
	

	return 0
}


func (self *Sshfs) Statfs(path string, stat *fuse.Statfs_t) (err int) {
	
	fmt.Printf("STAT FS!!! %s\n", path)
	

	// ssh sftp has StatVFS but ftp, github api, aws sdk might not
	info, _ := self.client.StatVFS(path)	
	stat.Bsize = info.Bsize
	stat.Frsize = info.Frsize
	stat.Blocks = info.Blocks
	stat.Bfree  = info.Bfree
	stat.Bavail = info.Bavail
	stat.Files  = info.Files
	stat.Ffree  = info.Ffree
	stat.Favail = info.Favail
	stat.Namemax = info.Namemax
	return 0
}


func main() {


	addr := "127.0.0.1:22"
	config := &ssh.ClientConfig{
		User: "user1",
		Auth: []ssh.AuthMethod{
			ssh.Password("password-here"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		//Ciphers: []string{"3des-cbc", "aes256-cbc", "aes192-cbc", "aes128-cbc"},
	}
	
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}
	
	client, err := sftp.NewClient(conn)
	if err != nil {
		panic("Failed to create client: " + err.Error())
	}
	
	cwd, err := client.Getwd()
	println("Current working directory:", cwd)



	sshfs := &Sshfs{}
	
	
	// init
	sshfs.client = client
	sshfs.nodes = make(map[string]*Node)
	
	
	
	host := fuse.NewFileSystemHost(sshfs)
	host.SetCapReaddirPlus(true)
	host.Mount("", append([]string{
		"-o", "ExactFileSystemName=NTFS",
		"-o", fmt.Sprintf("volname=%s", "Nice"),
	}, os.Args[1:]...))	
	
	
	// done
	client.Close()
}
