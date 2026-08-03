package app

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"

	"salestracker/internal/config"
	"salestracker/internal/handler"
	"salestracker/internal/repo"
	"salestracker/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

//go:embed web
var webFiles embed.FS

type App struct {
	srv *http.Server
	db  *sql.DB
}

func New(cfg *config.Config) *App {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	r := repo.New(db)
	if err := r.Migrate(); err != nil {
		log.Fatal(err)
	}

	svc := service.New(r)
	h := handler.New(svc)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.POST("/items", h.Create)
	router.GET("/items", h.List)
	router.PUT("/items/:id", h.Update)
	router.DELETE("/items/:id", h.Delete)
	router.GET("/items/export", h.ExportCSV)
	router.GET("/analytics", h.Analytics)

	fe, _ := fs.Sub(webFiles, "web")
	router.StaticFS("/static", http.FS(fe))
	router.GET("/", func(c *gin.Context) {
		data, _ := webFiles.ReadFile("web/index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	return &App{
		srv: &http.Server{Addr: cfg.ServerPort, Handler: router},
		db:  db,
	}
}

func (a *App) Run() error {
	log.Printf("listening on %s", a.srv.Addr)
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.db.Close()
	return a.srv.Shutdown(ctx)
}
