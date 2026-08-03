package app

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"

	"commenttree/internal/repository"
	"commenttree/internal/service"
	"commenttree/internal/transport"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

//go:embed all:../../frontend
var frontend embed.FS

type App struct {
	srv *http.Server
	db  *sql.DB
}

func New() *App {
	db, err := sql.Open("sqlite", "comments.db")
	if err != nil {
		log.Fatal(err)
	}

	if err := migrate(db); err != nil {
		log.Fatal(err)
	}

	repo := repository.New(db)
	svc := service.New(repo)
	handler := transport.New(svc)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	api := router.Group("/")
	{
		api.POST("/comments", handler.Create)
		api.GET("/comments", handler.Get)
		api.DELETE("/comments/:id", handler.Delete)
		api.GET("/search", handler.Search)
	}

	fe, _ := fs.Sub(frontend, "web")
	router.StaticFS("/static", http.FS(fe))
	router.GET("/", func(c *gin.Context) {
		data, _ := frontend.ReadFile("web/index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	return &App{
		srv: &http.Server{
			Addr:    ":8080",
			Handler: router,
		},
		db: db,
	}
}

func (a *App) Run() error {
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.srv.Shutdown(ctx)
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id INTEGER,
			path TEXT NOT NULL,
			body TEXT NOT NULL,
			author TEXT NOT NULL DEFAULT 'Anonymous',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_path ON comments(path);
		CREATE VIRTUAL TABLE IF NOT EXISTS comments_fts USING fts5(body, content=comments, content_rowid=rowid);
		CREATE TRIGGER IF NOT EXISTS comments_ai AFTER INSERT ON comments BEGIN
			INSERT INTO comments_fts(rowid, body) VALUES (new.rowid, new.body);
		END;
		CREATE TRIGGER IF NOT EXISTS comments_ad AFTER DELETE ON comments BEGIN
			INSERT INTO comments_fts(comments_fts, rowid, body) VALUES('delete', old.rowid, old.body);
		END;
		CREATE TRIGGER IF NOT EXISTS comments_au AFTER UPDATE ON comments BEGIN
			INSERT INTO comments_fts(comments_fts, rowid, body) VALUES('delete', old.rowid, old.body);
			INSERT INTO comments_fts(rowid, body) VALUES (new.rowid, new.body);
		END;
	`)
	return err
}
