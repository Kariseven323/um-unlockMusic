package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"go.uber.org/zap"

	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/internal/utils"
)

// FFmpegProcessPool FFmpeg进程池，用于复用进程减少启动开销
type FFmpegProcessPool struct {
	maxSize     int
	processes   chan *exec.Cmd
	mutex       sync.Mutex
	initialized bool
}

var (
	globalFFmpegPool *FFmpegProcessPool
	poolOnce         sync.Once
)

// GetFFmpegPool 获取全局FFmpeg进程池
func GetFFmpegPool() *FFmpegProcessPool {
	poolOnce.Do(func() {
		globalFFmpegPool = &FFmpegProcessPool{
			maxSize:   4, // 最多保持4个进程
			processes: make(chan *exec.Cmd, 4),
		}
	})
	return globalFFmpegPool
}

// getProcess 获取一个可用的FFmpeg进程（暂时简化，直接创建新进程）
func (p *FFmpegProcessPool) getProcess(ctx context.Context, args ...string) *exec.Cmd {
	// 简化实现：直接创建新进程
	// 完整的进程池实现需要更复杂的生命周期管理
	return exec.CommandContext(ctx, "ffmpeg", args...)
}

// getProbeProcess 获取一个可用的FFprobe进程
func (p *FFmpegProcessPool) getProbeProcess(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "ffprobe", args...)
}

func ExtractAlbumArt(ctx context.Context, rd io.Reader) (*bytes.Buffer, error) {
	pool := GetFFmpegPool()
	cmd := pool.getProcess(ctx,
		"-i", "pipe:0", // input from stdin
		"-an",              // disable audio
		"-codec:v", "copy", // copy video(image) codec
		"-f", "image2", // use image2 muxer
		"pipe:1", // output to stdout
	)

	cmd.Stdin = rd
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg run: %w", err)
	}

	return stdout, nil
}

type UpdateMetadataParams struct {
	Audio    string // required
	AudioExt string // required

	Meta common.AudioMeta // required

	AlbumArt    []byte // optional
	AlbumArtExt string // required if AlbumArt is not nil
}

func UpdateMeta(ctx context.Context, outPath string, params *UpdateMetadataParams, logger *zap.Logger) error {
	if params.AudioExt == ".flac" {
		return updateMetaFlac(ctx, outPath, params, logger.With(zap.String("module", "updateMetaFlac")))
	} else {
		return updateMetaFFmpeg(ctx, outPath, params)
	}
}
func updateMetaFFmpeg(ctx context.Context, outPath string, params *UpdateMetadataParams) error {
	builder := newFFmpegBuilder()

	out := newOutputBuilder(outPath) // output to file
	builder.SetFlag("y")             // overwrite output file
	builder.AddOutput(out)

	// input audio -> output audio
	builder.AddInput(newInputBuilder(params.Audio)) // input 0: audio
	out.AddOption("map", "0:a")
	out.AddOption("codec:a", "copy")

	// input cover -> output cover
	if params.AlbumArt != nil &&
		params.AudioExt != ".wav" /* wav doesn't support attached image */ {

		// write cover to temp file
		artPath, err := utils.WriteTempFile(bytes.NewReader(params.AlbumArt), params.AlbumArtExt)
		if err != nil {
			return fmt.Errorf("updateAudioMeta write temp file: %w", err)
		}
		defer os.Remove(artPath)

		builder.AddInput(newInputBuilder(artPath)) // input 1: cover
		out.AddOption("map", "1:v")

		switch params.AudioExt {
		case ".ogg": // ogg only supports theora codec
			out.AddOption("codec:v", "libtheora")
		case ".m4a": // .m4a(mp4) requires set codec, disposition, stream metadata
			out.AddOption("codec:v", "mjpeg")
			out.AddOption("disposition:v", "attached_pic")
			out.AddMetadata("s:v", "title", "Album cover")
			out.AddMetadata("s:v", "comment", "Cover (front)")
		case ".mp3":
			out.AddOption("codec:v", "mjpeg")
			out.AddMetadata("s:v", "title", "Album cover")
			out.AddMetadata("s:v", "comment", "Cover (front)")
		default: // other formats use default behavior
		}
	}

	// set file metadata
	album := params.Meta.GetAlbum()
	if album != "" {
		out.AddMetadata("", "album", album)
	}

	title := params.Meta.GetTitle()
	if title != "" {
		out.AddMetadata("", "title", title)
	}

	artists := params.Meta.GetArtists()
	if len(artists) != 0 {
		// Enhanced multi-artist support with format-specific handling
		setMultipleArtists(out, artists, params.AudioExt)
	}

	if params.AudioExt == ".mp3" {
		out.AddOption("write_id3v1", "true")
		out.AddOption("id3v2_version", "3")
	}

	// execute ffmpeg
	cmd := builder.Command(ctx)

	if stdout, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg run: %w, %s", err, string(stdout))
	}

	return nil
}

// setMultipleArtists sets artist metadata with format-specific optimizations
func setMultipleArtists(out *outputBuilder, artists []string, audioExt string) {
	if len(artists) == 0 {
		return
	}

	// For single artist, use standard approach
	if len(artists) == 1 {
		out.AddMetadata("", "artist", artists[0])
		return
	}

	// Format-specific multi-artist handling
	switch audioExt {
	case ".mp3":
		// ID3v2.3 supports multiple artist frames
		// Use semicolon separator for better compatibility
		out.AddMetadata("", "artist", strings.Join(artists, "; "))
		// Also set albumartist for better player support
		out.AddMetadata("", "albumartist", strings.Join(artists, "; "))

	case ".flac":
		// FLAC supports multiple ARTIST fields natively
		// This will be handled in meta_flac.go, use standard format here
		out.AddMetadata("", "artist", strings.Join(artists, "; "))

	case ".m4a", ".mp4":
		// MP4 supports multiple artists with proper atom structure
		// Use semicolon separator for better compatibility
		out.AddMetadata("", "artist", strings.Join(artists, "; "))
		out.AddMetadata("", "albumartist", strings.Join(artists, "; "))

	case ".ogg":
		// Vorbis comments support multiple ARTIST fields
		// Use semicolon separator for better compatibility
		out.AddMetadata("", "artist", strings.Join(artists, "; "))

	default:
		// For other formats, use semicolon separator (better than " / ")
		out.AddMetadata("", "artist", strings.Join(artists, "; "))
	}
}
