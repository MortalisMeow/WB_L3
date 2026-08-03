package app

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"

	"warehousecontrol/internal/auth"
	"warehousecontrol/internal/config"
	"warehousecontrol/internal/handler"
	"warehousecontrol/internal/middleware"
	"warehousecontrol/internal/models"
	"warehousecontrol/internal/repo"
	"warehousecontrol/internal/service"

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

	jwt := auth.New(cfg.JWTSecret)
	svc := service.New(r)
	h := handler.New(svc, jwt)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.POST("/auth/login", h.Login)

	api := router.Group("/")
	api.Use(middleware.Auth(jwt))

	api.GET("/items", middleware.RequireRoles(models.RoleAdmin, models.RoleManager, models.RoleViewer), h.List)
	api.GET("/items/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleManager, models.RoleViewer), h.Get)
	api.POST("/items", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), h.Create)
	api.PUT("/items/:id", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), h.Update)
	api.DELETE("/items/:id", middleware.RequireRoles(models.RoleAdmin), h.Delete)

	api.GET("/items/:id/history", middleware.RequireRoles(models.RoleAdmin, models.RoleManager, models.RoleViewer), h.ItemHistory)
	api.GET("/history/export", middleware.RequireRoles(models.RoleAdmin, models.RoleManager), h.ExportHistoryCSV)
	api.GET("/history", middleware.RequireRoles(models.RoleAdmin, models.RoleManager, models.RoleViewer), h.History)
	api.GET("/history/:id/diff", middleware.RequireRoles(models.RoleAdmin, models.RoleManager, models.RoleViewer), h.HistoryDiff)

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
