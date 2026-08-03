package app

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"

	"eventbooker/internal/api"
	"eventbooker/internal/repo"
	"eventbooker/internal/svc"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

//go:embed web
var webFiles embed.FS

type App struct {
	srv *http.Server
}

func New() *App {
	db, err := sql.Open("sqlite", "events.db")
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY,
		name TEXT,
		date TEXT,
		seats INTEGER,
		booked INTEGER DEFAULT 0
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS bookings (
		id INTEGER PRIMARY KEY,
		event_id INTEGER,
		user_name TEXT,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	r := repo.New(db)
	s := svc.New(r)
	go s.CancelWorker()
	h := api.New(s)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.POST("/events", h.CreateEvent)
	router.GET("/events", h.ListEvents)
	router.GET("/events/:id", h.GetEvent)
	router.POST("/events/:id/book", h.Book)
	router.POST("/events/:id/confirm", h.Confirm)
	router.DELETE("/events/:id", h.DeleteEvent)

	fe, _ := fs.Sub(webFiles, "web")
	router.StaticFS("/static", http.FS(fe))
	router.GET("/", func(c *gin.Context) {
		data, _ := webFiles.ReadFile("web/index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	return &App{
		srv: &http.Server{Addr: ":8080", Handler: router},
	}
}

func (a *App) Run() error {
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.srv.Shutdown(ctx)
}
