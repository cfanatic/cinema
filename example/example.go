package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jtguibas/cinema"
)

func main() {
	downloadTestVideo("example.mp4")

	video, err := cinema.Load("example.mp4")
	check(err)

	video.Trim(10*time.Second, 20*time.Second) // trim video from 10 to 20 seconds
	video.SetStart(1 * time.Second)            // trim first second of the video
	video.SetEnd(9 * time.Second)              // keep only up to 9 seconds
	video.SetSize(400, 300)                    // resize video to 400x300
	video.Crop(0, 0, 200, 200)                 // crop rectangle top-left (0,0) with size 200x200
	video.SetSize(400, 400)                    // resize cropped 200x200 video to a 400x400
	video.SetFPS(48)                           // set the output framerate to 48 frames per second
	video.SetBitrate(200_000)                  // set the output bitrate of 200 kbps
	video.Render("test_output1.mov")           // note format conversion by file extension

	// you can also generate the command line instead of applying it directly
	fmt.Println("FFMPEG Command", video.CommandLine("test_output1.mov"))

	// produce another test video and concatenate both clips
	video.Trim(42*time.Second, 48*time.Second)
	video.Render("test_output2.mov")
	clip, err := cinema.NewClip([]string{"test_output1.mov", "test_output2.mov"}) // absolute or relative paths can be used
	check(err)
	clip.Concatenate("concat.mov")
	fmt.Println("FFMPEG Command", clip.CommandLine("concat.mov"))
}

func downloadTestVideo(to string) {
	const url = "https://media.w3.org/2010/05/sintel/trailer.mp4"

	fmt.Println("downloading test video...")
	resp, err := http.Get(url)
	check(err)
	defer resp.Body.Close()

	out, err := os.Create(to)
	check(err)
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
