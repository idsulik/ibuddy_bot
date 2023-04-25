package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func convertOggToMp3(oggFilepath string) (mp3Filepath string, err error) {
	defer func() {
		os.Remove(oggFilepath)
	}()
	name := filepath.Base(oggFilepath)
	mp3Filepath = fmt.Sprintf("%s%s.mp3", os.TempDir(), name)

	params := []string{"-i", oggFilepath, "-ac", "1", mp3Filepath}
	cmd := exec.Command("ffmpeg", params...)

	if _, err = cmd.CombinedOutput(); err != nil {
		mp3Filepath = ""
	}

	return mp3Filepath, err
}
