package cinema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Video contains information about a video file and all the operations that
// need to be applied to it. Call Load to initialize a Video from file. Call the
// transformation functions to generate the desired output. Then call Render to
// generate the final output video file.
type Video struct {
	filepath       string
	width          int
	height         int
	fps            int
	bitrate        int
	start          time.Duration
	end            time.Duration
	duration       time.Duration
	filters        []string
	additionalArgs []string
}

// TODO
type Clip struct {
	videosPath       []string
	concatListCache string
}

// Load gives you a Video that can be operated on. Load does not open the file
// or load it into memory. Apply operations to the Video and call Render to
// generate the output video file.
func Load(path string) (*Video, error) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, errors.New("cinema.Load: ffprobe was not found in your PATH " +
			"environment variable, make sure to install ffmpeg " +
			"(https://ffmpeg.org/) and add ffmpeg, ffplay and ffprobe to your " +
			"PATH")
	}

	if _, err := os.Stat(path); err != nil {
		return nil, errors.New("cinema.Load: unable to load file: " + err.Error())
	}

	cmd := exec.Command(
		"ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	out, err := cmd.Output()

	if err != nil {
		return nil, errors.New("cinema.Load: ffprobe failed: " + err.Error())
	}

	type description struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
			Tags   struct {
				// Rotation is optional -> use a pointer.
				Rotation *json.Number `json:"rotate"`
			} `json:"tags"`
		} `json:"streams"`
		Format struct {
			DurationSec json.Number `json:"duration"`
			Bitrate     json.Number `json:"bit_rate"`
		} `json:"format"`
	}
	var desc description
	if err := json.Unmarshal(out, &desc); err != nil {
		return nil, errors.New("cinema.Load: unable to parse JSON output " +
			"from ffprobe: " + err.Error())
	}
	if len(desc.Streams) == 0 {
		return nil, errors.New("cinema.Load: ffprobe does not contain stream " +
			"data, make sure the file " + path + " contains a valid video.")
	}

	secs, err := desc.Format.DurationSec.Float64()
	if err != nil {
		return nil, errors.New("cinema.Load: ffprobe returned invalid duration: " +
			err.Error())
	}
	bitrate, err := desc.Format.Bitrate.Int64()
	if err != nil {
		return nil, errors.New("cinema.Load: ffprobe returned invalid duration: " +
			err.Error())
	}

	// Round seconds (floating point value) up to time.Duration. seconds will
	// be >= 0 so adding 0.5 rounds to the right integer Duration value.
	duration := time.Duration(secs*float64(time.Second) + 0.5)

	dsIndex := 0
	for index, v := range desc.Streams {
		if v.Width != 0 && v.Height != 0 {
			dsIndex = index
			break
		}
	}

	width := desc.Streams[dsIndex].Width
	height := desc.Streams[dsIndex].Height
	if desc.Streams[dsIndex].Tags.Rotation != nil {
		// If the video is rotated by -270, -90, 90 or 270 degrees, we need to
		// flip the width and height because they will be reported in unrotated
		// coordinates while cropping etc. works on the rotated dimensions.
		rotation, err := desc.Streams[dsIndex].Tags.Rotation.Int64()
		if err != nil {
			return nil, errors.New("cinema.Load: ffprobe returned invalid " +
				"rotation: " + err.Error())
		}
		flipCount := rotation / 90
		if flipCount%2 != 0 {
			width, height = height, width
		}
	}

	return &Video{
		filepath: path,
		width:    width,
		height:   height,
		fps:      30,
		bitrate:  int(bitrate),
		start:    0,
		end:      duration,
		duration: duration,
	}, nil
}

// Render applies all operations to the Video and creates an output video file
// of the given name. This method won't return anything on stdout / stderr.
// If you need to read ffmpeg's outputs, use RenderWithStreams
func (v *Video) Render(output string) error {
	return v.RenderWithStreams(output, nil, nil)
}

// RenderWithStreams applies all operations to the Video and creates an output video file
// of the given name. By specifying an output stream and an error stream, you can read
// ffmpeg's stdout and stderr.
func (v *Video) RenderWithStreams(output string, os io.Writer, es io.Writer) error {
	line := v.CommandLine(output)
	cmd := exec.Command(line[0], line[1:]...)
	cmd.Stderr = es
	cmd.Stdout = os

	err := cmd.Run()
	if err != nil {
		return errors.New("cinema.Video.Render: ffmpeg failed: " + err.Error())
	}
	return nil
}

// CommandLine returns the command line that will be used to convert the Video
// if you were to call Render.
func (v *Video) CommandLine(output string) []string {
	var filters string
	if len(v.filters) > 0 {
		filters = strings.Join(v.filters, ",") + ","
	}
	filters += "setsar=1,fps=fps=" + strconv.Itoa(int(v.fps))

	additionalArgs := v.additionalArgs

	cmdline := []string{
		"ffmpeg",
		"-y",
		"-i", v.filepath,
		"-ss", strconv.FormatFloat(v.start.Seconds(), 'f', -1, 64),
		"-t", strconv.FormatFloat((v.end - v.start).Seconds(), 'f', -1, 64),
		"-vb", strconv.Itoa(v.bitrate),
	}
	cmdline = append(cmdline, additionalArgs...)
	cmdline = append(cmdline, "-vf", filters, "-strict", "-2")
	cmdline = append(cmdline, output)
	return cmdline
}

// Mute mutes the video
func (v *Video) Mute() {
	v.additionalArgs = append(v.additionalArgs, "-an")
}

// Trim sets the start and end time of the output video. It is always relative
// to the original input video. start must be less than or equal to end or
// nothing will change.
func (v *Video) Trim(start, end time.Duration) {
	if start <= end {
		v.SetStart(start)
		v.SetEnd(end)
	}
}

// Start returns the start of the video .
func (v *Video) Start() time.Duration {
	return v.start
}

// SetStart sets the start time of the output video. It is always relative to
// the original input video.
func (v *Video) SetStart(start time.Duration) {
	v.start = v.clampToDuration(start)
	if v.start > v.end {
		// keep c.start <= v.end
		v.end = v.start
	}
}

func (v *Video) clampToDuration(t time.Duration) time.Duration {
	if t < 0 {
		t = 0
	}
	if t > v.duration {
		t = v.duration
	}
	return t
}

// End returns the end of the video.
func (v *Video) End() time.Duration {
	return v.end
}

// SetEnd sets the end time of the output video. It is always relative to the
// original input video.
func (v *Video) SetEnd(end time.Duration) {
	v.end = v.clampToDuration(end)
	if v.end < v.start {
		// keep c.start <= v.end
		v.start = v.end
	}
}

// SetFPS sets the framerate (frames per second) of the output video.
func (v *Video) SetFPS(fps int) {
	v.fps = fps
}

// SetBitrate sets the bitrate of the output video.
func (v *Video) SetBitrate(bitrate int) {
	v.bitrate = bitrate
}

// SetSize sets the width and height of the output video.
func (v *Video) SetSize(width int, height int) {
	v.width = width
	v.height = height
	v.filters = append(v.filters, fmt.Sprintf("scale=%d:%d", width, height))
}

// Width returns the width of the video in pixels.
func (v *Video) Width() int {
	return v.width
}

// Height returns the width of the video in pixels.
func (v *Video) Height() int {
	return v.height
}

// Crop makes the output video a sub-rectangle of the input video. (0,0) is the
// top-left of the video, x goes right, y goes down.
func (v *Video) Crop(x, y, width, height int) {
	v.width = width
	v.height = height
	v.filters = append(
		v.filters,
		fmt.Sprintf("crop=%d:%d:%d:%d", width, height, x, y),
	)
}

// Filepath returns the path of the input video.
func (v *Video) Filepath() string {
	return v.filepath
}

// Duration returns the duration of the original input video. It does not
// account for any trim operation (Trim, SetStart, SetEnd).
// To get the current trimmed duration use
//     v.End() - v.Start()
func (v *Video) Duration() time.Duration {
	return v.duration
}

// FPS returns the set fps of the current video struct
func (v *Video) FPS() int {
	return v.fps
}

// Bitrate returns the set bitrate of the current video struct
func (v *Video) Bitrate() int {
	return v.bitrate
}

// TODO
func NewClip(videoPath []string) (*Clip, error) {
	var clip Clip
	dir := filepath.Dir(videoPath[0])
	clip = Clip{videosPath: videoPath, concatListCache: filepath.Join(dir, "concat.txt")}
	return &clip, nil
}

// TODO
func (c *Clip) Concatenate(output string) error {
	return c.ConcatenateWithStreams(output, nil, nil)
}

// TODO
func (c *Clip) ConcatenateWithStreams(output string, os io.Writer, es io.Writer) error {
	c.saveConcatenateList()
	defer c.deleteConcatenateList()
	line := c.CommandLine(output)
	cmd := exec.Command(line[0], line[1:]...)
	cmd.Stderr = es
	cmd.Stdout = os

	err := cmd.Run()
	if err != nil {
		return errors.New("cinema.Video.Concatenate: ffmpeg failed: " + err.Error())
	}
	return nil
}

// TODO
func (c *Clip) CommandLine(output string) []string {
	cmdline := []string{
		"ffmpeg",
		"-y",
		"-f", "concat",
		"-i", c.concatListCache,
		"-c", "copy",
	}
	cmdline = append(cmdline, "-fflags", "+genpts", filepath.Join(filepath.Dir(c.videosPath[0]), output))
	return cmdline
}

func (c *Clip) saveConcatenateList() error {
	f, err := os.Create(c.concatListCache)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, video := range c.videosPath {
		fmt.Fprintf(f, "file '%s'\n", filepath.Base(video))
	}
	return nil
}

func (c *Clip) deleteConcatenateList() error {
	if err := os.Remove(c.concatListCache); err != nil {
		return err
	}
	return nil
}
