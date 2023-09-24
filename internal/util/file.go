package util

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func DownloadFileByUrl(url string) (*os.File, error) {
	file, err := os.CreateTemp(os.TempDir(), "voice")

	if err != nil {
		return nil, err
	}
	defer file.Close()

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)

	return file, err
}

func ConvertOggToMp3(oggFilepath string) (mp3Filepath string, err error) {
	name := filepath.Base(oggFilepath)
	mp3Filepath = fmt.Sprintf("%s%s.mp3", os.TempDir(), name)

	params := []string{"-i", oggFilepath, "-ac", "1", mp3Filepath}
	cmd := exec.Command("ffmpeg", params...)

	if _, err = cmd.CombinedOutput(); err != nil {
		mp3Filepath = ""
	}

	return mp3Filepath, err
}
