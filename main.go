package main

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	uuid "github.com/satori/go.uuid"
)

var theSession *session.Session

// GetConfig Initiatilize config in singleton way
func GetSession() *session.Session {
	if theSession == nil {
		theSession = initSession()
	}
	return theSession
}

func initSession() *session.Session {
	newSession := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("AWS_REGION"), // example: "ap-southeast-1"
		Credentials: credentials.NewStaticCredentials("ACCESS_KEY_ID", "ACCESS_KEY_SECRET", ""),
	}))
	return newSession
}

type UploadResult struct {
	Path string `json:"path" xml:"path"`
}

// Helper
func UploadToS3(c echo.Context, fileName string, src multipart.File) (string, error) {
	// The session the S3 Uploader will use
	sess := GetSession()
	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)
	// Upload the file to S3.
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      aws.String("fauziawsbucket"),
		Key:         aws.String(fileName),
		Body:        src,
		ContentType: aws.String("image/png"),
	})
	// content type penting agar saat link dibuka file tidak langsung auto download

	if err != nil {
		return "", fmt.Errorf("failed to upload file, %v", err)
	}
	return result.Location, nil
}

func main() {
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}, error=${error}\n",
	}))
	e.POST("/upload", func(c echo.Context) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{"message": "input file error"})
		}
		// karena saat upload file aws tidak generate nama file secara manual, sehingga harus generate nama filenya secara manual
		// gunakan package github.com/satori/go.uuid lalu panggil fungsinya uuid.NewV4().String()
		fileName := uuid.NewV4().String()
		file.Filename = fileName + file.Filename[(len(file.Filename)-5):len(file.Filename)]
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{"message": "error when open file"})
		}
		defer src.Close()

		uploadURL, err := UploadToS3(c, file.Filename, src)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{"message": "internal server error"})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"path_url": uploadURL,
			"message":  "succes uploaded",
		})
	})
	if err := e.Start(":8000"); err != nil {
		log.Println(err.Error())
	}
}
