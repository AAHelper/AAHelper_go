package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/AAHelper/AAHelper_go/models"
	"github.com/alexandrevicenzi/unchained"
	csrf "github.com/utrack/gin-csrf"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const UserID = "userid"

//AuthRequired to view private parts of the site
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userid := session.Get(UserID).(int64)
		user := models.User{}
		if err := db.Where(&models.User{ID: userid}).First(&user).Error; err != nil {
			// c.Redirect(302, "/login/")
			c.Redirect(302, "/login/")
		} else {
			// Continue down the chain to handler etc
			c.Set("user", user)
			c.Next()
		}
	}
}

func login(c *gin.Context) {
	session := sessions.Default(c)
	username := c.PostForm("username")
	password := c.PostForm("password")

	user := models.User{}
	// rows, err := conn.Model(&models.User{}).Select("area, slug").Order("id desc").Rows()
	// if err := query.Find(&meetings).Error; err != nil {
	// 	log.Fatal(err)
	// }
	if err := db.Where(&models.User{Username: strings.Trim(username, "")}).First(&user).Error; err != nil {
		c.Set("template", "templates/auth/login/login.html")
		c.Set("data", map[string]interface{}{
			"error":      "Username or password is incorrect",
			"csrf_token": csrf.GetToken(c),
		})
		c.Status(301)
	}

	// if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Parameters can't be empty"})
	// 	return
	// }
	valid, _ := unchained.CheckPassword(password, user.Password)

	if valid {
		session.Set(UserID, user.ID)
		err := session.Save()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token"})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
		}
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
	}
}

func logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
	} else {
		log.Println(user)
		session.Delete("user")
		session.Save()
		c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
	}
}

func alcholicIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"hello": c.MustGet("user").(models.User)})
}

func alcholicUserEcho(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"hello": "Logged in user"})
}