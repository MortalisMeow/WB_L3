package app

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"imageprocessor/internal/repository"
	"imageprocessor/internal/service"
	"imageprocessor/internal/transport"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

//go:embed frontend
var webFiles embed.FS

type App struct {
	srv    *http.Server
	db     *sql.DB
	writer *kafka.Writer
	reader *kafka.Reader
}

func getBroker() string {
	broker := os.Getenv("KAFKA_BROKERS")
	if broker == "" {
		broker = "localhost:9092"
	}
	return broker
}

func New() *App {
	os.MkdirAll("uploads", 0755)
	os.MkdirAll("processed", 0755)

	db, err := sql.Open("sqlite", "images.db")
	if err != nil {
		log.Fatal(err)
	}

	if err := migrate(db); err != nil {
		log.Fatal(err)
	}

	broker := getBroker()
	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    "image-processing",
		Balancer: &kafka.LeastBytes{},
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   "image-processing",
		GroupID: "image-processor-group",
	})

	repo := repository.New(db)
	svc := service.New(repo, writer, reader)

	go svc.StartWorker()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	handler := transport.New(svc)

	api := router.Group("/")
	{
		api.POST("/upload", handler.Upload)
		api.GET("/image/:id", handler.GetImage)
		api.GET("/status/:id", handler.GetStatus)
		api.DELETE("/image/:id", handler.Delete)
		api.GET("/images", handler.List)
	}

	fe, _ := fs.Sub(webFiles, "web")
	router.StaticFS("/static", http.FS(fe))
	router.Static("/processed", "./processed")
	router.GET("/", func(c *gin.Context) {
		data, _ := webFiles.ReadFile("web/index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	return &App{
		srv: &http.Server{
			Addr:    ":8080",
			Handler: router,
		},
		db:     db,
		writer: writer,
		reader: reader,
	}
}

func (a *App) Run() error {
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.writer.Close()
	a.reader.Close()
	return a.srv.Shutdown(ctx)
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS images (
			id INTEGER PRIMARY KEY,
			filename TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}
