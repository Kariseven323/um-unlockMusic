package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"unlock-music.dev/cli/algo/common"
	_ "unlock-music.dev/cli/algo/kgm"
	_ "unlock-music.dev/cli/algo/kwm"
	_ "unlock-music.dev/cli/algo/ncm"
	"unlock-music.dev/cli/algo/qmc"
	_ "unlock-music.dev/cli/algo/tm"
	_ "unlock-music.dev/cli/algo/xiami"
	_ "unlock-music.dev/cli/algo/ximalaya"
	"unlock-music.dev/cli/internal/cache"
	"unlock-music.dev/cli/internal/ffmpeg"
	"unlock-music.dev/cli/internal/pool"
	"unlock-music.dev/cli/internal/sniff"
	"unlock-music.dev/cli/internal/utils"
)

var AppVersion = "custom"

func main() {
	module, ok := debug.ReadBuildInfo()
	if ok && module.Main.Version != "(devel)" {
		AppVersion = module.Main.Version
	}
	app := cli.App{
		Name:     "Unlock Music CLI",
		HelpName: "um",
		Usage:    "Unlock your encrypted music file https://git.unlock-music.dev/um/cli",
		Version:  fmt.Sprintf("%s (%s,%s/%s)", AppVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "input", Aliases: []string{"i"}, Usage: "path to input file or dir", Required: false},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "path to output dir", Required: false},
			&cli.StringFlag{Name: "qmc-mmkv", Aliases: []string{"db"}, Usage: "path to qmc mmkv (.crc file also required)", Required: false},
			&cli.StringFlag{Name: "qmc-mmkv-key", Aliases: []string{"key"}, Usage: "mmkv password (16 ascii chars)", Required: false},
			&cli.StringFlag{Name: "kgg-db", Usage: "path to kgg db (win32 kugou v11)", Required: false},
			&cli.BoolFlag{Name: "remove-source", Aliases: []string{"rs"}, Usage: "remove source file", Required: false, Value: false},
			&cli.BoolFlag{Name: "skip-noop", Aliases: []string{"n"}, Usage: "skip noop decoder", Required: false, Value: true},
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"V"}, Usage: "verbose logging", Required: false, Value: false},
			&cli.BoolFlag{Name: "update-metadata", Usage: "update metadata & album art from network", Required: false, Value: false},
			&cli.BoolFlag{Name: "overwrite", Usage: "overwrite output file without asking", Required: false, Value: false},
			&cli.BoolFlag{Name: "watch", Usage: "watch the input dir and process new files", Required: false, Value: false},
			&cli.BoolFlag{Name: "batch", Usage: "batch processing mode (read JSON from stdin)", Required: false, Value: false},
			&cli.BoolFlag{Name: "service", Usage: "run as service mode (IPC communication)", Required: false, Value: false},
			&cli.StringFlag{Name: "service-pipe", Usage: "service pipe name (Windows) or socket path (Unix)", Required: false, Value: ""},
			&cli.StringFlag{Name: "naming-format", Usage: "output filename format: auto (smart detection), title-artist, artist-title, original", Required: false, Value: "auto"},

			&cli.BoolFlag{Name: "supported-ext", Usage: "show supported file extensions and exit", Required: false, Value: false},
		},

		Action:          appMain,
		Copyright:       fmt.Sprintf("Copyright (c) 2020 - %d Unlock Music https://git.unlock-music.dev/um/cli/src/branch/master/LICENSE", time.Now().Year()),
		HideHelpCommand: true,
		UsageText:       "um [-o /path/to/output/dir] [--extra-flags] [-i] /path/to/input",
	}

	err := app.Run(os.Args)
	if err != nil {
		// Use a temporary logger for fatal errors in main
		tempLogger := setupLogger(false)
		tempLogger.Fatal("run app failed", zap.Error(err))
	}
}

func printSupportedExtensions() {
	var exts []string
	extSet := make(map[string]int)
	for _, factory := range common.DecoderRegistry {
		ext := strings.TrimPrefix(factory.Suffix, ".")
		if n, ok := extSet[ext]; ok {
			extSet[ext] = n + 1
		} else {
			extSet[ext] = 1
		}
	}
	for ext := range extSet {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	for _, ext := range exts {
		fmt.Printf("%s: %d\n", ext, extSet[ext])
	}
}

func setupLogger(verbose bool) *zap.Logger {
	logConfig := zap.NewProductionEncoderConfig()
	logConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	enabler := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		if verbose {
			return true
		}
		return level >= zapcore.InfoLevel
	})

	return zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(logConfig),
		os.Stderr,
		enabler,
	))
}

func appMain(c *cli.Context) (err error) {
	logger := setupLogger(c.Bool("verbose"))

	// 检查是否为服务模式
	if c.Bool("service") {
		pipeName := c.String("service-pipe")
		return runServiceMode(logger, pipeName)
	}

	// 检查是否为批处理模式
	if c.Bool("batch") {
		return runBatchMode(logger)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if c.Bool("supported-ext") {
		printSupportedExtensions()
		return nil
	}
	input := c.String("input")
	if input == "" {
		switch c.Args().Len() {
		case 0:
			input = cwd
		case 1:
			input = c.Args().Get(0)
		default:
			return errors.New("please specify input file (or directory)")
		}
	}

	input, absErr := filepath.Abs(input)
	if absErr != nil {
		return fmt.Errorf("get abs path failed: %w", absErr)
	}

	output := c.String("output")
	inputStat, err := os.Stat(input)
	if err != nil {
		return err
	}

	var inputDir string
	if inputStat.IsDir() {
		inputDir = input
	} else {
		inputDir = filepath.Dir(input)
	}
	inputDir, absErr = filepath.Abs(inputDir)
	if absErr != nil {
		return fmt.Errorf("get abs path (inputDir) failed: %w", absErr)
	}

	if output == "" {
		// Default to where the input dir is
		output = inputDir
	}
	logger.Debug("resolve input/output path", zap.String("inputDir", inputDir), zap.String("input", input), zap.String("output", output))

	outputStat, err := os.Stat(output)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = os.MkdirAll(output, 0755)
		}
		if err != nil {
			return err
		}
	} else if !outputStat.IsDir() {
		return errors.New("output should be a writable directory")
	}

	if mmkv := c.String("qmc-mmkv"); mmkv != "" {
		// If key is not set, the mmkv vault will be treated as unencrypted.
		key := c.String("qmc-mmkv-key")
		err := qmc.OpenMMKV(mmkv, key, logger)
		if err != nil {
			return err
		}
	}

	kggDbPath := c.String("kgg-db")
	if kggDbPath == "" {
		kggDbPath = filepath.Join(os.Getenv("APPDATA"), "Kugou8", "KGMusicV3.db")
	}

	proc := &processor{
		logger:          logger,
		inputDir:        inputDir,
		outputDir:       output,
		kggDbPath:       kggDbPath,
		skipNoopDecoder: c.Bool("skip-noop"),
		removeSource:    c.Bool("remove-source"),
		updateMetadata:  c.Bool("update-metadata"),
		overwriteOutput: c.Bool("overwrite"),
		namingFormat:    c.String("naming-format"),
	}

	if inputStat.IsDir() {
		watchDir := c.Bool("watch")
		if !watchDir {
			return proc.processDir(input)
		} else {
			return proc.watchDir(input)
		}
	} else {
		return proc.processFile(input)
	}

}

type processor struct {
	logger    *zap.Logger
	inputDir  string
	outputDir string

	kggDbPath string

	skipNoopDecoder bool
	removeSource    bool
	updateMetadata  bool
	overwriteOutput bool
	namingFormat    string
}

// generateOutputFilename 根据命名格式生成输出文件名
func (p *processor) generateOutputFilename(inputFilename, audioExt string) string {
	switch p.namingFormat {
	case "original":
		// 保持原文件名不变
		return inputFilename + audioExt
	case "title-artist":
		// 强制使用"歌曲名-歌手名"格式
		return p.formatFilename(inputFilename, audioExt, true)
	case "artist-title":
		// 强制使用"歌手名-歌曲名"格式
		return p.formatFilename(inputFilename, audioExt, false)
	case "auto":
		fallthrough
	default:
		// 使用智能解析，保持原文件的命名方式
		return p.formatFilenameAuto(inputFilename, audioExt)
	}
}

// formatFilenameAuto 使用智能解析格式化文件名，保持原文件的命名方式
func (p *processor) formatFilenameAuto(inputFilename, audioExt string) string {
	// 使用智能解析获取元数据
	meta := common.SmartParseFilenameMeta(inputFilename)

	title := meta.GetTitle()
	artists := meta.GetArtists()
	originalFormat := meta.GetOriginalFormat()

	// 如果解析失败，回退到原文件名
	if title == "" {
		return inputFilename + audioExt
	}

	// 构建艺术家字符串
	var artistStr string
	if len(artists) > 0 {
		artistStr = strings.Join(artists, ", ")
	}

	// 根据原始格式决定输出格式
	switch originalFormat {
	case "title-artist":
		// 原文件是"歌曲名-歌手名"格式
		if artistStr != "" {
			return title + " - " + artistStr + audioExt
		}
	case "artist-title":
		// 原文件是"歌手名-歌曲名"格式
		if artistStr != "" {
			return artistStr + " - " + title + audioExt
		}
	case "title-only", "empty":
		// 只有标题或空，直接使用标题
		return title + audioExt
	default:
		// 未知格式，使用默认的"歌手名-歌曲名"格式
		if artistStr != "" {
			return artistStr + " - " + title + audioExt
		}
	}

	// 只有标题，没有艺术家信息
	return title + audioExt
}

// formatFilename 使用智能解析格式化文件名
func (p *processor) formatFilename(inputFilename, audioExt string, titleFirst bool) string {
	// 使用智能解析获取元数据
	meta := common.SmartParseFilenameMeta(inputFilename)

	title := meta.GetTitle()
	artists := meta.GetArtists()

	// 如果解析失败，回退到原文件名
	if title == "" {
		return inputFilename + audioExt
	}

	// 构建艺术家字符串
	var artistStr string
	if len(artists) > 0 {
		artistStr = strings.Join(artists, ", ")
	}

	// 根据格式要求生成文件名
	if titleFirst && artistStr != "" {
		// 歌曲名-歌手名格式
		return title + " - " + artistStr + audioExt
	} else if !titleFirst && artistStr != "" {
		// 歌手名-歌曲名格式
		return artistStr + " - " + title + audioExt
	} else {
		// 只有标题，没有艺术家信息
		return title + audioExt
	}
}

func (p *processor) watchDir(inputDir string) error {
	if err := p.processDir(inputDir); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
					// try open with exclusive mode, to avoid file is still writing
					f, err := os.OpenFile(event.Name, os.O_RDONLY, os.ModeExclusive)
					if err != nil {
						p.logger.Debug("failed to open file exclusively", zap.String("path", event.Name), zap.Error(err))
						time.Sleep(1 * time.Second) // wait for file writing complete
						continue
					}
					_ = f.Close()

					if err := p.processFile(event.Name); err != nil {
						p.logger.Warn("failed to process file", zap.String("path", event.Name), zap.Error(err))
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				p.logger.Error("file watcher got error", zap.Error(err))
			}
		}
	}()

	err = watcher.Add(inputDir)
	if err != nil {
		return fmt.Errorf("failed to watch dir %s: %w", inputDir, err)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	<-signalCtx.Done()
	return nil
}

func (p *processor) processDir(inputDir string) error {
	items, err := os.ReadDir(inputDir)
	if err != nil {
		return err
	}

	var lastError error = nil
	for _, item := range items {
		filePath := filepath.Join(inputDir, item.Name())
		if item.IsDir() {
			if err = p.processDir(filePath); err != nil {
				lastError = err
			}
			continue
		}

		if err := p.processFile(filePath); err != nil {
			lastError = err
			p.logger.Error("conversion failed", zap.String("source", item.Name()), zap.Error(err))
		}
	}
	if lastError != nil {
		return fmt.Errorf("last error: %w", lastError)
	}
	return nil
}

func (p *processor) processFile(filePath string) error {
	p.logger.Debug("processFile", zap.String("file", filePath), zap.String("inputDir", p.inputDir))

	allDec := common.GetDecoder(filePath, p.skipNoopDecoder)
	if len(allDec) == 0 {
		return errors.New("skipping while no suitable decoder")
	}

	if err := p.process(filePath, allDec); err != nil {
		return err
	}

	// if source file need to be removed
	if p.removeSource {
		err := os.RemoveAll(filePath)
		if err != nil {
			return err
		}
		p.logger.Info("source file removed after success conversion", zap.String("source", filePath))
	}
	return nil
}

func (p *processor) findDecoder(decoders []common.DecoderFactory, params *common.DecoderParams) (*common.Decoder, *common.DecoderFactory, error) {
	for _, factory := range decoders {
		dec := factory.Create(params)
		err := dec.Validate()
		if err == nil {
			return &dec, &factory, nil
		}
		p.logger.Warn("try decode failed", zap.Error(err))
	}
	return nil, nil, errors.New("no any decoder can resolve the file")
}

func (p *processor) process(inputFile string, allDec []common.DecoderFactory) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	logger := p.logger.With(zap.String("source", inputFile))

	pDec, decoderFactory, err := p.findDecoder(allDec, &common.DecoderParams{
		Reader:          file,
		Extension:       filepath.Ext(inputFile),
		FilePath:        inputFile,
		Logger:          logger,
		KggDatabasePath: p.kggDbPath,
	})
	if err != nil {
		return err
	}
	dec := *pDec

	params := &ffmpeg.UpdateMetadataParams{}

	// 使用内存池获取header缓冲区，增加大小以提高格式识别准确性
	headerBuf := pool.GetBuffer(256) // 从64字节增加到256字节
	defer pool.PutBuffer(headerBuf)

	_, err = io.ReadFull(dec, headerBuf)
	if err != nil {
		return fmt.Errorf("read header failed: %w", err)
	}
	header := bytes.NewBuffer(headerBuf)
	audio := io.MultiReader(header, dec)
	// 使用智能fallback，根据输入文件扩展名推测输出格式
	inputExt := filepath.Ext(inputFile)
	params.AudioExt = sniff.AudioExtensionWithSmartFallback(header.Bytes(), inputExt)
	logger.Info("format detection", zap.String("inputExt", inputExt), zap.String("detectedExt", params.AudioExt))

	if p.updateMetadata {
		if audioMetaGetter, ok := dec.(common.AudioMetaGetter); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// since ffmpeg doesn't support multiple input streams,
			// we need to write the audio to a temp file.
			// Optimized: use streaming approach for QMC decoder with Seek support
			params.Audio, err = p.createOptimizedTempFile(dec, audio, params.AudioExt)
			if err != nil {
				return fmt.Errorf("updateAudioMeta write temp file: %w", err)
			}
			defer os.Remove(params.Audio)

			params.Meta, err = audioMetaGetter.GetAudioMeta(ctx)
			if err != nil {
				logger.Warn("get audio meta failed", zap.Error(err))
			}

			if params.Meta == nil { // reset audio meta if failed
				audio, err = os.Open(params.Audio)
				if err != nil {
					return fmt.Errorf("updateAudioMeta open temp file: %w", err)
				}
			}
		}
	}

	// 检查元数据缓存
	var cachedEntry *cache.MetadataEntry
	if p.updateMetadata {
		if stat, err := os.Stat(inputFile); err == nil {
			cachedEntry, _ = cache.GetMetadata(inputFile, stat.Size(), stat.ModTime())
		}
	}

	// 用文件名信息包装元数据，确保标题准确性
	if p.updateMetadata {
		params.Meta = p.processMetadataWithFilename(params.Meta, inputFile, cachedEntry, logger)
	} else {
		// 即使没有启用update-metadata，也使用文件名信息生成基本元数据
		// 这样可以保留文件名中的重要信息（如Live、Remix等标识）
		params.Meta = common.SmartParseFilenameMeta(filepath.Base(inputFile))
	}

	if p.updateMetadata && params.Meta != nil {
		if coverGetter, ok := dec.(common.CoverImageGetter); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if cover, err := coverGetter.GetCoverImage(ctx); err != nil {
				logger.Warn("get cover image failed", zap.Error(err))
			} else if imgExt, ok := sniff.ImageExtension(cover); !ok {
				logger.Warn("sniff cover image type failed", zap.Error(err))
			} else {
				params.AlbumArtExt = imgExt
				params.AlbumArt = cover
			}
		}
	}

	inputRelDir, err := filepath.Rel(p.inputDir, filepath.Dir(inputFile))
	if err != nil {
		return fmt.Errorf("get relative dir failed: %w", err)
	}

	inFilename := strings.TrimSuffix(filepath.Base(inputFile), decoderFactory.Suffix)
	outFilename := p.generateOutputFilename(inFilename, params.AudioExt)
	outPath := filepath.Join(p.outputDir, inputRelDir, outFilename)

	if !p.overwriteOutput {
		_, err := os.Stat(outPath)
		if err == nil {
			logger.Warn("output file already exist, skip", zap.String("destination", outPath))
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat output file failed: %w", err)
		}
	}

	if params.Meta == nil {
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer outFile.Close()

		if _, err := utils.OptimizedCopy(outFile, audio); err != nil {
			return err
		}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		if err := ffmpeg.UpdateMeta(ctx, outPath, params, logger); err != nil {
			return err
		}
	}

	logger.Info("successfully converted", zap.String("source", inputFile), zap.String("destination", outPath))
	return nil
}

// processMetadataWithFilename 处理元数据并用文件名信息包装
func (p *processor) processMetadataWithFilename(rawMeta common.AudioMeta, inputFile string, cachedEntry *cache.MetadataEntry, logger *zap.Logger) common.AudioMeta {
	if cachedEntry != nil {
		// 使用缓存的元数据
		logger.Debug("使用缓存的元数据", zap.String("文件", inputFile))
		return cachedEntry.Meta
	}

	if rawMeta != nil {
		// 用文件名信息包装原始元数据
		wrappedMeta := common.WrapMetaWithFilename(rawMeta, filepath.Base(inputFile))

		// 缓存元数据
		if stat, err := os.Stat(inputFile); err == nil {
			cache.PutMetadata(inputFile, stat.Size(), stat.ModTime(), wrappedMeta, nil)
		}

		return wrappedMeta
	}

	// 如果元数据获取失败，直接使用文件名元数据
	return common.SmartParseFilenameMeta(filepath.Base(inputFile))
}

// createOptimizedTempFile creates a temporary file using optimized approach
// For QMC decoders with Seek support, this avoids reading the entire file into memory
func (p *processor) createOptimizedTempFile(dec common.Decoder, fallbackReader io.Reader, ext string) (string, error) {
	// Check if decoder supports seeking (like our optimized QMC decoder)
	if seeker, ok := dec.(io.Seeker); ok {
		// Reset to beginning for streaming
		if _, err := seeker.Seek(0, io.SeekStart); err == nil {
			// Use the decoder directly as it supports seeking
			return utils.OptimizedWriteTempFile(dec, ext)
		}
	}

	// Fallback to the original approach for non-seeking decoders
	return utils.OptimizedWriteTempFile(fallbackReader, ext)
}
