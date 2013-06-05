package pogoutils

import (
	"time"
	"os"
	"fmt"
	"net/http"
	"io"
)

//This function allows you to determine if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

//This function returns the time.Time that a file was last modified at
func LastMod(path string) (time.Time) {
	fileInfo, _ := os.Stat(path)
	return fileInfo.ModTime()
}

//Download a file from the given URL and save it to the given file
//Note that the Instagram API encourages you to take into account the IP of Instagram
//users, so you shouldn't download files with this
func Download(url, saveFile string) {
	fmt.Println("Downloading", url, "to", saveFile)
	out, err := os.Create(saveFile)
	if err != nil {
		return
	}
	defer out.Close()
	
	resp, err := http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		io.Copy(out, resp.Body)
	}
	fmt.Println("Downloaded", url, "to", saveFile)
}

//Create a folder at the given URL
func CreateFolder(url string) {
	os.Mkdir(url, 0777)
}