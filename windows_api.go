package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

const IoctlVolumeGetVolumeDiskExtents uint32 = 0x00560000

// A Volume could be on many physical drives.
// Returns a list of string containing each physical drive the volume uses.
// For CD Drives with no disc in it will return an empty list.
func drivetoextents(driveletter string) ([]DiskExtent, error) {
	drive, err := syscall.CreateFile(syscall.StringToUTF16Ptr(`\\.\`+driveletter+`:`),
		syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil,
		syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		// l.WarningF(9999, "Could not open drive %v: %v", driveletter, err)
		return []DiskExtent{}, err
	}
	var bytesreturned uint32
	var extents DiskExtents
	err = syscall.DeviceIoControl(drive, IoctlVolumeGetVolumeDiskExtents, nil, 0,
		(*byte)(unsafe.Pointer(&extents)), uint32(unsafe.Sizeof(extents)), (*uint32)(unsafe.Pointer(&bytesreturned)), nil)
	runtime.KeepAlive(bytesreturned)

	if err != nil {
		return []DiskExtent{}, err
	}
	return extents.Extents[0:extents.NumberOfExtents], nil
}

type DiskExtent struct {
	DiskNumber     uint32
	StartingOffset uint64
	ExtentLength   uint64
}

type DiskExtents struct {
	NumberOfExtents uint32
	Padding         uint32
	Extents         [128]DiskExtent
}

func getFileOffsets(targetfiles []string) ([]fileinfo, []error) {
	var results []fileinfo
	var errors []error
	for _, target := range targetfiles {
		result, err := getFileOffset(target)
		if err != nil {
			errors = append(errors, err)
		} else {
			results = append(results, result)
		}
	}
	return results, errors
}

const (
	FILE_READ_ATTRIBUTES uint32 = 0x80
	SYNCHRONIZE          uint32 = 0x100000
)

func getFileOffset(target string) (result fileinfo, rerr error) {
	namep, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		rerr = err
		return
	}

	fh, err := syscall.CreateFile(namep, FILE_READ_ATTRIBUTES|SYNCHRONIZE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		rerr = err
		return
	}
	defer syscall.CloseHandle(fh)

	stat, rerr := os.Stat(target)
	if rerr != nil {
		rerr = err
		return
	}
	if stat.IsDir() {
		rerr = fmt.Errorf("Target %v is a directory", target)
		return
	}

	offset, err := partitionoffset(fh, 0)
	if err != nil {
		rerr = err
	}

	if offset == 0xFFFFFFFFFFFFFFFF {
		// Maybe it's compressed
		namep, err = syscall.UTF16PtrFromString(target + ":WofCompressedData:$DATA")
		if err != nil {
			rerr = err
			return
		}

		fh, err := syscall.CreateFile(namep, FILE_READ_ATTRIBUTES|SYNCHRONIZE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
		if err != nil {
			rerr = err
			return
		}

		defer syscall.CloseHandle(fh)

		offset, err = partitionoffset(fh, 0)
		if err != nil {
			rerr = err
			return
		}
	}

	result = fileinfo{
		filename:  target,
		offset:    offset,
		drivename: `\\.\` + filepath.VolumeName(target),
	}
	return
}

const (
	FSCTL_GET_RETRIEVAL_POINTERS uint32 = 0x00090073 //https://msdn.microsoft.com/en-us/library/cc246805.aspx
)

type extent struct {
	nextvcn uint64
	lcn     uint64
}

type retrieval_pointers_buffer struct {
	extentcount uint32
	padding     uint32
	startingvcn uint64
	extents     [16384]extent
}

func partitionoffset(fh syscall.Handle, cluster_offset uint64) (uint64, error) {
	var starting_vcn uint64
	var outbuffer retrieval_pointers_buffer
	var bytesreturned uint32
	starting_vcn = cluster_offset

	err := syscall.DeviceIoControl(fh,
		FSCTL_GET_RETRIEVAL_POINTERS,
		(*byte)(unsafe.Pointer(&starting_vcn)), // A pointer to the input buffer, a STARTING_VCN_INPUT_BUFFER structure. (LPVOID)
		uint32(unsafe.Sizeof(starting_vcn)),    // The size of the input buffer, in bytes. (DWORD)
		(*byte)(unsafe.Pointer(&outbuffer)),    // A pointer to the output buffer, a RETRIEVAL_POINTERS_BUFFER variably sized structure (LPVOID)
		uint32(unsafe.Sizeof(outbuffer)),       // The size of the input buffer, in bytes. (DWORD)
		&bytesreturned,                         // A pointer to a variable that receives the size of the data stored in the output buffer, in bytes. (LPDWORD)
		nil)                                    // lpOverlapped  //A pointer to an OVERLAPPED structure; if fd is opened without specifying FILE_FLAG_OVERLAPPED, lpOverlapped is ignored.(LPOVERLAPPED)

	if err != nil {
		// if err.Error() != "More data is available." {
		return 0, err
	}

	if bytesreturned == 0 {
		return 0, errors.New("No data returned")
	}

	// vcn := outbuffer.startingvcn
	// for i := 0; i < int(outbuffer.extentcount); i++ {
	// 	if i > 0 {
	// 		vcn = outbuffer.extents[i-1].nextvcn
	// 	}
	// 	fmt.Printf("Got extent %v: VCN %016X at LCN %016X\n", i, vcn, outbuffer.extents[i].lcn)
	// }

	return uint64(outbuffer.extents[0].lcn), nil
}
