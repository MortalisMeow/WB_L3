package service

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"imageprocessor/internal/models"
	"imageprocessor/internal/repository"

	"github.com/nfnt/resize"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type TaskMessage struct {
	ID       int64  `json:"id"`
	Filename string `json:"filename"`
}

type Service struct {
	repo   *repository.Repo
	writer *kafka.Writer
	reader *kafka.Reader
}

func New(repo *repository.Repo, writer *kafka.Writer, reader *kafka.Reader) *Service {
	return &Service{repo: repo, writer: writer, reader: reader}
}

func (s *Service) Upload(filename string, data []byte) (*models.Image, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		return nil, fmt.Errorf("unsupported format: %s", ext)
	}

	savePath := filepath.Join("uploads", filename)
	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return nil, err
	}

	img, err := s.repo.Create(filename)
	if err != nil {
		return nil, err
	}

	msg := TaskMessage{ID: img.ID, Filename: img.Filename}
	msgData, _ := json.Marshal(msg)

	s.writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", img.ID)),
		Value: msgData,
	})

	return img, nil
}

func (s *Service) GetImage(id int64) (*models.Image, error) {
	return s.repo.GetByID(id)
}

func (s *Service) Delete(id int64) error {
	img, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	os.Remove(filepath.Join("uploads", img.Filename))
	os.Remove(filepath.Join("processed", "thumb_"+img.Filename))
	os.Remove(filepath.Join("processed", "wm_"+img.Filename))

	return s.repo.Delete(id)
}

func (s *Service) List() ([]*models.Image, error) {
	return s.repo.List()
}

func (s *Service) StartWorker() {
	for {
		msg, err := s.reader.ReadMessage(context.Background())
		if err != nil {
			continue
		}

		var task TaskMessage
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			continue
		}

		s.processImage(task)
	}
}

func (s *Service) processImage(task TaskMessage) {
	s.repo.UpdateStatus(task.ID, "processing")

	srcPath := filepath.Join("uploads", task.Filename)
	srcFile, err := os.Open(srcPath)
	if err != nil {
		s.repo.UpdateStatus(task.ID, "failed")
		return
	}
	defer srcFile.Close()

	srcImg, _, err := image.Decode(srcFile)
	if err != nil {
		s.repo.UpdateStatus(task.ID, "failed")
		return
	}

	thumb := resize.Resize(150, 0, srcImg, resize.Lanczos3)
	thumbPath := filepath.Join("processed", "thumb_"+task.Filename)
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		s.repo.UpdateStatus(task.ID, "failed")
		return
	}
	jpeg.Encode(thumbFile, thumb, &jpeg.Options{Quality: 80})
	thumbFile.Close()

	wmImg := addWatermark(srcImg, "SAMPLE")
	wmPath := filepath.Join("processed", "wm_"+task.Filename)
	wmFile, err := os.Create(wmPath)
	if err != nil {
		s.repo.UpdateStatus(task.ID, "failed")
		return
	}
	defer wmFile.Close()

	ext := strings.ToLower(filepath.Ext(task.Filename))
	if ext == ".png" {
		png.Encode(wmFile, wmImg)
	} else {
		jpeg.Encode(wmFile, wmImg, &jpeg.Options{Quality: 85})
	}

	s.repo.UpdateStatus(task.ID, "done")
}

func addWatermark(img image.Image, text string) image.Image {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(image.Black),
		Face: basicfont.Face7x13,
		Dot: fixed.Point26_6{
			X: fixed.I(bounds.Max.X - len(text)*7 - 10),
			Y: fixed.I(bounds.Max.Y - 10),
		},
	}
	d.DrawString(text)

	return rgba
}
