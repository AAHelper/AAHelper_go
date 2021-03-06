package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	// "github.com/gin-contrib/pprof"
	_ "github.com/heroku/x/hmetrics/onload"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/joho/godotenv/autoload"

	// "github.com/ngerakines/ginpongo2"
	csrf "github.com/utrack/gin-csrf"
)

//BinaryFS struct
type BinaryFS struct {
	fs http.FileSystem
}

//Open opens a file with a name
func (b *BinaryFS) Open(name string) (http.File, error) {
	return b.fs.Open(name)
}

//Exists checks if a filepath with the prefix exists
func (b *BinaryFS) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		if _, err := b.fs.Open(p); err != nil {
			return false
		}
		return true
	}
	return false
}

//BFS for our static files.
func BFS(root string) *BinaryFS {
	fs := &assetfs.AssetFS{
		Asset:     Asset,
		AssetDir:  AssetDir,
		AssetInfo: AssetInfo,
		Prefix:    root}
	return &BinaryFS{
		fs,
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}
	// conn := getEnv("PG_CONN", )
	conn := os.Getenv("PG_CONN")

	if conn == "" {
		log.Fatal("PG_CONN not set.")
	}

	secret := os.Getenv("SECRET_KEY")
	csrfSecret := os.Getenv("CSRF_SECRET_KEY")

	if secret == "" {
		secret = "WhaTtwaSstHhepeRrsonthinkIingwHhentheYyDdiscoveredcOow’smIilkWwasfinEefoRrhumAancoNnsumption…AandwhYyDdidtheYydOoIitIinthEefirsTtPplace!?"
	}
	if csrfSecret == "" {
		csrfSecret = "HEerAanouTtOofmoneYy,sOoHhehAadTtostoPpplayIingPpoker!."
	}

	db, err := gorm.Open("postgres", conn)

	if err != nil {
		panic(err)
	}
	if gin.IsDebugging() {
		db.LogMode(true)
	}

	defer db.Close()

	r := gin.Default()

	store := cookie.NewStore([]byte(secret))
	session := sessions.Sessions("mysession", store)
	r.Use(session)
	r.Use(csrf.Middleware(csrf.Options{
		Secret: csrfSecret,
		ErrorFunc: func(c *gin.Context) {
			c.String(400, "CSRF token mismatch")
			c.Abort()
		},
	}))

	// to rebuild the bfs tree run
	// go-bindata -ignore '.DS*' ./static/...
	bfs := BFS("static")
	r.Use(static.Serve("/static", BFS("static")))
	r.GET("/favicon.ico", func(c *gin.Context) {
		fileserver := http.FileServer(bfs)
		r2 := new(http.Request)
		*r2 = *c.Request
		r2.URL = new(url.URL)
		*r2.URL = *c.Request.URL
		r2.URL.Path = "/favicons/favicon.ico"

		fileserver.ServeHTTP(c.Writer, r2)
		c.Abort()
	})

	r.GET("/", Pongo2(), func(c *gin.Context) {
		index(db, c, false)
	})
	r.POST("/", Pongo2(), func(c *gin.Context) {
		index(db, c, true)
	})
	r.GET("/locations/:id", Pongo2(), func(c *gin.Context) {
		id := c.Param("id")
		locationID, _ := strconv.ParseInt(id, 10, 64)
		locationDetail(locationID, db, c)
	})
	r.GET("/area/:slug", Pongo2(), func(c *gin.Context) {
		Slug := c.Param("slug")
		areaDetail(Slug, db, c)
	})

	e := r.Run(":" + port)
	if e != nil {
		panic("Could not run :(")
	}
}
