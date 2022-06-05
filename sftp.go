/*
 * hellofs.go
 *
 * Copyright 2017-2022 Bill Zissimopoulos
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
	"github.com/winfsp/cgofuse/fuse"
	//"time"
	//"log"


	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)





type Sftpfs struct {
	fuse.FileSystemBase
	client *sftp.Client
}













func (self *Sftpfs) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {

	fmt.Printf("(+) Getattr %s\n", path)
	
	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 0777
		return 0	
	} else {
	
		info, err := self.client.Stat(path)
		if err != nil {
			fmt.Println(err)
			return -fuse.ENOENT
		}
		
		if info.IsDir() == true {
			stat.Mode = fuse.S_IFDIR | 0777
			return 0		
		} else {
			stat.Mode = fuse.S_IFREG | 0777
			stat.Size = info.Size()
			return 0		
		}
	}
	
	return -fuse.ENOENT
}

func (self *Sftpfs) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {

	//fmt.Printf("(+) Read %s\n", path)

	//endofst := ofst + int64(len(buff))
	//if endofst > int64(len(contents)) {
	//	endofst = int64(len(contents))
	//}
	//if endofst < ofst {
	//	return 0
	//}
	//n = copy(buff, contents[ofst:endofst])
	return
}

func (self *Sftpfs) Readdir(path string, fill func(name string, stat *fuse.Stat_t, ofst int64) bool, ofst int64, fh uint64) (errc int) {
	
	fmt.Printf("(+) Readdir %s\n", path)
	
	fill(".", nil, 0)
	fill("..", nil, 0)
	
	// update nodes
	entries, err := self.client.ReadDir(path)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, entry := range entries {
			fill(entry.Name(), nil, 0)
		}
	}
	
	return 0
}

func main() {

	sftpfs := &Sftpfs{}

	addr := "127.0.0.1"
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

	sftpfs.client = client


	host := fuse.NewFileSystemHost(sftpfs)
	host.SetCapReaddirPlus(true)
	host.Mount("", append([]string{
		"-o", "ExactFileSystemName=NTFS",
		"-o", fmt.Sprintf("volname=%s", "Nice"),
	}, os.Args[1:]...))	
	
	
	// done
	client.Close()
}