package main

import (
	"context"
	"log"
	"net/http"
	neturl "net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ctx    = context.TODO()
	C, e   = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	Client = C.Database("gintodo")
	rdb    = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
)

func init() {
	if e != nil {
		panic(e)
	}
	e = C.Ping(ctx, nil)
	if e != nil {
		panic(e)
	}
}

func Error(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

type Url struct {
	Link string `json:"link" form:"link" binding:"required"`
}

func IsUrl(str string) bool {
	u, err := neturl.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func createShortened(c *gin.Context) {
	var url Url
	if err := c.ShouldBindJSON(&url); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !IsUrl(url.Link) || !strings.HasPrefix(url.Link, "https://") && !strings.HasPrefix(url.Link, "http://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad url"})
		return
	}

	if id, err := rdb.Get(ctx, url.Link).Result(); err == redis.Nil {
		newid, err := gonanoid.New(5)
		Error(err)
		_, _ = Client.Collection("urls").InsertOne(ctx, bson.D{primitive.E{Key: "id", Value: newid}, primitive.E{Key: "link", Value: url.Link}})
		err = rdb.Set(ctx, url.Link, newid, 0).Err()
		Error(err)
		c.JSON(200, gin.H{"id": newid})
	} else {
		c.JSON(200, gin.H{"id": id})
	}
}

func getUrl(c *gin.Context) {
	var res Url
	id := c.Param("id")
	Client.Collection("urls").FindOne(ctx, bson.D{primitive.E{Key: "id", Value: id}}).Decode(&res)
	if (Url{}) != res {
		c.JSON(200, res)
	} else {
		c.JSON(404, gin.H{"error": "Not Found"})
	}
}

func main() {
	r := gin.Default()
	r.POST("/api/create", createShortened)
	r.GET("/api/:id", getUrl)
	r.Run()
}
