# cinema : a lightweight video editing library for Go

![alt text](https://i.imgur.com/uYRpL29.jpg "github.com/jtguibas/cinema")

## Overview [![GoDoc](https://godoc.org/github.com/cfanatic/cinema?status.svg)](https://godoc.org/github.com/cfanatic/cinema)

cinema is a simple video editing library based on ffmpeg. It supports trimming, resizing, cropping, concatenation and more. Use it to create videos directly or let it generate command lines that use ffmpeg for you.

## Installation

You must have [FFMPEG](https://ffmpeg.org/download.html) installed on your machine! Make sure `ffmpeg` and `ffprobe` are available from the command line on your machine.
To install `cinema` run:

```bash
go get github.com/jtguibas/cinema
```

## Example Usage

```golang
func main() {
    downloadTestVideo("example.mp4")

    video, err := cinema.Load("example.mp4")
    heck(err)

    video.Trim(10*time.Second, 20*time.Second) // trim video from 10 to 20 seconds
    video.SetStart(1 * time.Second)            // trim first second of the video
    video.SetEnd(9 * time.Second)              // keep only up to 9 seconds
    video.SetSize(400, 300)                    // resize video to 400x300
    video.Crop(0, 0, 200, 200)                 // crop rectangle top-left (0,0) with size 200x200
    video.SetSize(400, 400)                    // resize cropped 200x200 video to a 400x400
    video.SetFPS(48)                           // set the output framerate to 48 frames per second
    video.Render("test_output.mov")            // note format conversion by file extension

    // you can also generate the command line instead of applying it directly
    fmt.Println("FFMPEG Command", video.CommandLine("test_output.mov"))

    // produce another test video and concatenate both clips
    video.Trim(42*time.Second, 48*time.Second)
    video.Render("test_output2.mov")
    clip, err := cinema.NewClip([]string{"test_output1.mov", "test_output2.mov"})
    check(err)
    clip.Concatenate("concat.mov")
    fmt.Println("FFMPEG Command", clip.CommandLine("concat.mov"))
}
```

## TODO

- [ ] expand to audio
- [ ] implement bitrate support
- [x] add concatenation support
- [x] improve godoc documentation
- [x] add cropping support
- [x] test ubuntu support
- [x] implement fps support

Feel free to open pull requests!

## Acknowledgments

- Big thanks to [gonutz](https://github.com/gonutz) for contributing to this project!
