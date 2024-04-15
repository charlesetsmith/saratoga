// See if a file exists or can be opened on the system
package fileio

import (
	"fmt"
	"os"
	"time"

	"github.com/charlesetsmith/saratoga/sarflags"
	"github.com/charlesetsmith/saratoga/sarsys"
)

// Check to see if a file or directory exists on our local system
// no named pipes or symbolic links supported at this time
func FileExists(fname string) bool {
	fileinfo, err := os.Stat(fname)
	if err == nil && (fileinfo.Mode().IsDir() || fileinfo.Mode().IsRegular()) {
		return true
	}
	return false
}

// Open a file on our local system
func FileOpen(fname string, ttype string) (*os.File, error) {
	var err error
	var fp *os.File
	switch ttype {
	case "get", "getremove": // We are getting a remote file
		// Open up to write to a local file
		// Dont stomp on any existing file
		if FileExists(fname) {
			return nil, fmt.Errorf("file already exists:  %s", fname)
		}
		// Create the file
		if fp, err = os.Create(fname); err == nil {
			return fp, nil
		}
		return nil, err
	case "getdir": // We are getting all the files from a remote directory
		if FileExists(fname) {
			return nil, fmt.Errorf("directory already exists: %s", fname)
		}
		// Create the Directory
		// BUT THEN WHAT DO WE DO!!!!
		// LOTS CODE TO WRITE HERE
		err = os.Mkdir(fname, os.ModeDir)
		if err == nil {
			if err = os.Chdir(fname); err != nil {
				return nil, err
			}
		}
		return nil, err
	case "put", "putdelete": // We are putting a local file to a remote system
		// Open up to read from the local file
		if !FileExists(fname) {
			return nil, fmt.Errorf("file does not exist: %s", fname)
		}
		if fp, err = os.Open(fname); err == nil {
			return fp, nil
		}
		return nil, err
	case "delete": // We are deleting a file from a remote system. We dont do that here!
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid transfer type %s", ttype)
	}
}

// Seek to a position from the origin in the file
func FileSeek(fp *os.File, pos uint64) error {
	const origin = 0 // Offset is always from the begining of the file
	var ipos, npos int64
	var err error

	if fp == nil {
		return fmt.Errorf("fileSeek:  file not open")
	}
	ipos = int64(pos) // Yep the Maximum File Size in go is MaxInt64 NOT MaxUint64
	// We have jumped pass MaxInt (gone -ve)
	if ipos < 0 {
		return fmt.Errorf("fileWrite: pos (%d) is > MaxUint64 (%d)", pos, sarflags.MaxInt64)
	}
	// Bad Seek
	if npos, err = fp.Seek(ipos, origin); err != nil || npos != ipos {
		return err
	}
	return nil
}

func FileRead(fp *os.File, pos uint64, blen int) ([]byte, error) {
	var err error
	var n int
	buf := make([]byte, blen)

	if fp == nil {
		return nil, fmt.Errorf("fileRead: file is closed")
	}

	// Seek to where we want to go
	if err = FileSeek(fp, pos); err != nil {
		return nil, err
	}
	if n, err = fp.Read(buf); err != nil {
		return nil, fmt.Errorf("fileread: only read %d, should be %d", n, blen)
	}
	buf = buf[0:n]
	return buf, nil
}

// Seek to a position in the file and write out the buffer to it
func FileWrite(fp *os.File, pos uint64, buf []byte) (int, error) {
	var n int
	var err error

	if err = FileSeek(fp, pos); err != nil {
		return 0, err
	}

	// We have written something but there is an issue
	if n, err = fp.Write(buf); err != nil {
		return n, err
	}
	// All good n can be <= len(buf)
	return n, err
}

// Close an open fp
func FileClose(fp *os.File) error {
	return fp.Close()
}

// Just Zap the file
func FileRm(fname string) error {
	return os.Remove(fname)
}

// Close the fp and then delete the file
func FileDelete(fp *os.File) error {

	if fp != nil {
		fname := fp.Name()
		FileClose(fp)
		return FileRm(fname)
	}
	return fmt.Errorf("no existing open file to delete")
}

// File Information Summary
type FileMetaData struct {
	Path        string      // File Name
	Origin      string      // Origin File Name for Symymbolic Links
	Mode        os.FileMode // Mode rwx etc.
	Info        os.FileInfo // Summary of Info in case you want to use this
	Size        int64       // Size of the File
	ModTime     time.Time   // Modification Time
	IsDir       bool        // Are we a directory or
	IsRegular   bool        // are we a regular file or
	IsNamedPipe bool        // are we a named pipe or
	IsSymLink   bool        // are we a symbolic link
	Uid         int         // Uid (0 for Windows)
	Gid         int         // Gid (0 for Windows)
}

// FileMeta - Get file metadata information
func FileMeta(filePath string) (*FileMetaData, error) {

	var fs *FileMetaData = new(FileMetaData)

	var err error
	var info os.FileInfo
	if info, err = os.Lstat(filePath); os.IsNotExist(err) {
		return nil, err
	}
	// Symbolic Links and Named Pipes are treated as "special files"
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		fs.IsSymLink = true
		fs.Origin, _ = os.Readlink(fs.Path)
		var patherr error
		if fs.Origin, patherr = os.Readlink(filePath); patherr != nil {
			return nil, patherr
		}
		origstat, _ := os.Lstat(fs.Origin)
		// Yes we can be a symbolic link to a directory so both are true
		fs.IsDir = origstat.IsDir()
	} else {
		fs.IsDir = info.IsDir()
	}
	if info.Mode()&os.ModeNamedPipe == os.ModeNamedPipe {
		fs.IsNamedPipe = true
	}
	fs.Info = info
	fs.Mode = info.Mode()
	fs.Path = info.Name()
	fs.Size = info.Size()
	fs.Uid, fs.Gid = sarsys.GetOwnerAndGroup(info) // 0, 0 for Windows of course...
	fs.ModTime = info.ModTime()
	fs.IsRegular = info.Mode().IsRegular()
	return fs, nil
}
