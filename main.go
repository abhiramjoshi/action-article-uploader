package main

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	githubactions "github.com/sethvargo/go-githubactions"
)
var logger *slog.Logger

type Image struct {
	Filename string `json:"filename"`
	Data     *string `json:"data"`
}

type Article struct {
	Title   string  `json:"title"`
	Path    string  `json:"-"`
	Images  []Image `json:"images"`
	Content string  `json:"content"`
}

func init() {
  env := os.Getenv("ENV")
  if len(env) == 0 {
    env = "PROD"
  }

  var level slog.Level
  if env == "DEV" {
    level = slog.LevelDebug
  } else {
    level = slog.LevelWarn
  }
  handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: level,
  })
  logger = slog.New(handler)
}

func main() {
  var folder string
  action := githubactions.New()
  if os.Getenv("ENV") == "DEV" {
    folder = "test"
  }else{
    folder = action.GetInput("article_folder")
  }
	articleName, articleFilepath, articlePhotos, err := parseArticle(folder)
	if err != nil {
		logger.Error("There was an error parsing the article folder","error", err)
	}
	article, _ := createArticlePayload(articleName, articleFilepath, articlePhotos)
	if err != nil {
    logger.Error("There was an error creating the article payload","error",err)
	}
	response, err := sendPostRequest(article)
	if err != nil {
		logger.Error("There was an error sending the post request","error", err)
	}
	defer response.Body.Close()
  body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Error("Error reading response body","error", err)
	}
  logger.Info("Recieved response","status", *&response.Status ,"body", string(body))
	logger.Debug(fmt.Sprintf("%v, %v, %v, %v", folder, articleFilepath, articleName, articlePhotos))
}

// Read markdown files from folder that is passed through
func parseArticle(articleFolder string) (string, string, string, error) {

	// check if folder exists
	info, err := os.Stat(articleFolder)
	if os.IsNotExist(err) {
		return "", "", "", fmt.Errorf("Specified article folder %v does not exist", articleFolder)
	}

	// check if folder is a directory
	if !info.IsDir() {
		return "", "", "", fmt.Errorf("Specified article folder %v is not a directory", articleFolder)
	}

	folderFiles, err := os.ReadDir(articleFolder)
	if err != nil {
		panic(err)
	}

	var articleFile string = ""
	var articleName string = ""

	for _, file := range folderFiles {
		logger.Debug(fmt.Sprintf("Considering file %v in article folder", file.Name()))
		logger.Debug(fmt.Sprintf("File extension is: %v", filepath.Ext(file.Name())))
		if filepath.Ext(file.Name()) == ".md" {
			articleFile = file.Name()
			articleName = strings.TrimRight(file.Name(), ".md")
		}
	}

	if articleFile == "" {
		return "", "", "", fmt.Errorf("Folder must contain an article written as a `.md` file")
	}

	articleFilepath := filepath.Join(articleFolder, articleFile)

	articlePhotos := filepath.Join(articleFolder, "photos")
	if _, err := os.Stat(articlePhotos); os.IsNotExist(err) {
		articlePhotos = ""
	}

	return articleName, articleFilepath, articlePhotos, nil
}

// Send POST request to site
func sendPostRequest(article Article) (*http.Response, error) {
	base_url, exists := os.LookupEnv("BASE_DOMAIN")
	if !exists {
		return nil, fmt.Errorf("Base url does not exists, please set the BASE_DOMAIN env variable")
	}
	post_endpoint, exists := os.LookupEnv("ENDPOINT")
  if !exists {
    return nil, fmt.Errorf("Endpoint not provided, please set the ENDPOINT env variable")
  }
  url := "http://" + base_url + "/" + post_endpoint
  logger.Debug(fmt.Sprintf("Sending request to: %v", url))
	// Need to handle authentication, this can be just simple authentication in our case.
	user := os.Getenv("USERNAME")
	pass := os.Getenv("PASSWORD")
  auth := user + ":" + pass
	basicAuth := b64.StdEncoding.EncodeToString([]byte(auth))

	body, err := json.Marshal(article)
	if err != nil {
		return nil, fmt.Errorf("There was an error formatting the post body: %v", err)
	}
	logger.Debug(fmt.Sprintf("Article post request: %v", string(body)))
	// send post request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	req.Header.Add("Authorization", "Basic "+basicAuth)
	req.Header.Add("Content-Type", "application/json")
  resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending article creation request: %v", err)
	}
  
	return resp, nil
}

func createArticlePayload(articleName string, articleFile string, articlePhotos string) (Article, error) {
	// Creates an article object that can be sent via a POST requests
	data, err := os.ReadFile(articleFile)
	if err != nil {
		return Article{}, fmt.Errorf("Error reading file: %v", err)
	}

	content := strings.ReplaceAll(string(data), "\r\n", " ")
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.TrimSpace(content)
	//Parse images
	imageFiles, err := os.ReadDir(articlePhotos)
	if err != nil {
		return Article{}, err
	}
	var images []Image
	for _, image := range imageFiles {
		ext := filepath.Ext(image.Name())
		imageName := strings.TrimRight(image.Name(), ext)
		images = append(images, Image{Filename: imageName, Data: nil})
	}

	return Article{Title: articleName, Content: content, Images: images, Path: filepath.Dir(articleFile)}, nil
}
