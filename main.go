package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/AAHelper/AAHelper_go/models"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/flosch/pongo2"
	"github.com/getsentry/raven-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/joho/godotenv/autoload"
	"github.com/ngerakines/ginpongo2"
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

var db *gorm.DB

func connectToOrSetDbConnection() {
	// conn := getEnv("PG_CONN", )
	if db == nil {
		conn := os.Getenv("PG_CONN")

		if conn == "" {
			log.Fatal("PG_CONN not set.")
		}
		var err error
		db, err = gorm.Open("postgres", conn)
		if err != nil {
			log.Fatal("Could not connect to database.")
		}
	}
}

func init() {
	sentryEnv := os.Getenv("SENTRY_ENV")
	if sentryEnv == "" {
		sentryEnv = "development"
	}
	connectToOrSetDbConnection()
	raven.SetDSN("https://36e41022dc29476bbeb4632af557d3a1:baa65a1b92424132b412973b50954803@sentry.io/1391009")
	raven.SetEnvironment("staging")
}

//UserMiddleware adds a user to the gin context
func PongoContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			user = models.User{}
		}
		context := map[string]interface{}{
			"csrf_token": csrf.GetToken(c),
			"user":       user,
		}
		c.Set("context", context)
		c.Next()
	}
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	secret := os.Getenv("SECRET_KEY")
	csrfSecret := os.Getenv("CSRF_SECRET_KEY")

	if secret == "" {
		secret = "WhaTtwaSstHhepeRrsonthinkIingwHhentheYyDdiscoveredcOow’smIilkWwasfinEefoRrhumAancoNnsumption…AandwhYyDdidtheYydOoIitIinthEefirsTtPplace!?"
	}
	if csrfSecret == "" {
		csrfSecret = "HEerAanouTtOofmoneYy,sOoHhehAadTtostoPpplayIingPpoker!."
	}

	connectToOrSetDbConnection()

	if gin.IsDebugging() {
		db.LogMode(true)
	}

	defer db.Close()

	r := gin.Default()
	// pongo2.NewLocalFileSystemLoader
	loader, err := pongo2.NewLocalFileSystemLoader("./templates")
	if err != nil {
		panic(err)
	}
	MyLoader := loader
	MySet := pongo2.NewSet("default", MyLoader)
	pongo2.FromFile = MySet.FromFile

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
	r.Use(static.Serve("/static", BFS("static")))
	r.GET("/favicon.ico", favIconDotICO)

	r.GET("/", ginpongo2.Pongo2(), UserMiddleWare(), PongoContextMiddleware(), indexGet)
	r.POST("/", ginpongo2.Pongo2(), UserMiddleWare(), PongoContextMiddleware(), indexPost)
	r.GET("/locations/:id", ginpongo2.Pongo2(), UserMiddleWare(), PongoContextMiddleware(), locationsById)
	r.GET("/area/:slug", ginpongo2.Pongo2(), UserMiddleWare(), PongoContextMiddleware(), areaSlugDetail)

	//Authentication
	r.GET("/login", ginpongo2.Pongo2(), login)
	r.POST("/login", ginpongo2.Pongo2(), login)
	r.GET("/logout", ginpongo2.Pongo2(), logout)
	private := r.Group("/alcholic")
	{
		private.GET("/", ginpongo2.Pongo2(), UserMiddleWare(), alcholicIndex)
		private.GET("/two", ginpongo2.Pongo2(), UserMiddleWare(), alcholicUserEcho)
	}
	private.Use(AuthRequired())

	e := r.Run(":" + port)
	if e != nil {
		panic("Could not run :(")
	}
}

func favIconDotICO(c *gin.Context) {
	bfs := BFS("static")
	fileserver := http.FileServer(bfs)
	r2 := new(http.Request)
	*r2 = *c.Request
	r2.URL = new(url.URL)
	*r2.URL = *c.Request.URL
	r2.URL.Path = "/favicons/favicon.ico"

	fileserver.ServeHTTP(c.Writer, r2)
	c.Abort()
}

func indexGet(c *gin.Context) {
	index(db, c, false)
}
func indexPost(c *gin.Context) {
	index(db, c, true)
}

func locationsById(c *gin.Context) {
	id := c.Param("id")
	locationID, _ := strconv.ParseInt(id, 10, 64)
	locationDetail(locationID, db, c)
}

func areaSlugDetail(c *gin.Context) {
	Slug := c.Param("slug")
	areaDetail(Slug, db, c)
}
