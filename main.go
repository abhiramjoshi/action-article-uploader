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
	Filename string  `json:"filename"`
	Data     *string `json:"data"`
	Filepath string  `json:"-"`
}

type Article struct {
	ID      *int    `json:"id,omitempty"`
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
	// Get article folder
	var folder string
	action := githubactions.New()
	if os.Getenv("PLATFORM") != "GITHUB" {
		folder = "test"
	} else {
		folder = action.GetInput("article_folder")
		logger.Debug(fmt.Sprintf("Getting input from Github Input: %v", folder))
	}
	// Parse article folder to create article payload
	articleName, articleFilepath, articlePhotos, err := parseArticle(folder)
	if err != nil {
		logger.Error("There was an error parsing the article folder", "error", err)
		os.Exit(1)
  }
  article, err := createArticlePayload(articleName, articleFilepath, articlePhotos)
  if err != nil {
    logger.Error("There was an error creating the article payload", "error", err)
    os.Exit(1)
  }
  if os.Getenv("DRYRUN") == "true" {
    logger.Debug("Not sending POST request in dry run")
    os.Exit(0)
  }
  // Check if article exists
  existArticle, err := checkIfArticleExists(article)
  if err != nil {
    logger.Error("There was an error checking if article exists", "error", err)
    os.Exit(1)
  }

  // If exists, send put request
  var response *http.Response
  if existArticle != nil {
    logger.Debug("Article exists, so sending PATCH")
		response, err = sendPutRequest(article, *existArticle.ID)
	} else {
    logger.Debug("Article does not exist, so sending POST")
		response, err = sendPostRequest(article)
	}
	if err != nil {
		logger.Error("There was an error sending the post request", "error", err)
		os.Exit(1)
	}
	// If not exists, send post request
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Error("Error reading response body", "error", err)
		os.Exit(1)
	}
	logger.Info("Recieved response", "status", *&response.Status, "body", string(body))
	fmt.Printf("Recieved response: status: %v body: %v", response.Status, string(body))
	logger.Debug(fmt.Sprintf("%v, %v, %v, %v", folder, articleFilepath, articleName, articlePhotos))
}

func checkIfImage(imageData []byte) bool {
	mimeType := http.DetectContentType(imageData)
	logger.Info(fmt.Sprintf("Image is of type: %v", mimeType))
	imageMimes := []string{"image/jpeg", "image/png", "image/gif"}
	for _, imageType := range imageMimes {
		if mimeType == imageType {
			return true
		}
	}
	return false
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
		return "", "", "", err
	}

	var articleFile string = ""
	var articleName string = ""

	for _, file := range folderFiles {
		logger.Debug(fmt.Sprintf("Considering file %v in article folder", file.Name()))
		logger.Debug(fmt.Sprintf("File extension is: %v", filepath.Ext(file.Name())))
		if filepath.Ext(file.Name()) == ".md" {
			articleFile = file.Name()
			articleName = strings.TrimSuffix(file.Name(), ".md")
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

func checkIfArticleExists(article Article) (*Article, error) {
	// Send get request to check if article exists
	base_url, exists := os.LookupEnv("BASE_DOMAIN")
	if !exists {
		return nil, fmt.Errorf("Base url does not exists, please set the BASE_DOMAIN env variable")
	}
	get_endpoint, exists := os.LookupEnv("GET_ENDPOINT")
	if !exists {
		return nil, fmt.Errorf("Endpoint not provided, please set the ENDPOINT env variable")
	}

	env := os.Getenv("ENV")
	if len(env) == 0 {
		env = "PROD"
	}
	protocol := "https://"
	if env == "DEV" {
		protocol = "http://"
	}

	url := protocol + base_url + "/" + get_endpoint
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("There was an error requesting articles")
		return nil, err
	}
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("There was an issue reading the response body")
		return nil, err
	}

	var articles []Article
	json.Unmarshal(response, &articles)
  //logger.Debug("Returned articles", "articles", fmt.Sprintf("%v", articles))
	for i := 0; i < len(articles); i++ {
		logger.Debug("Checking articles for matches", "article1", article, "article2", articles[i])
    if article.Title == articles[i].Title {
			logger.Debug("Article titles matches", article.Title, articles[i].Title)
      return &articles[i], nil
		}
	}
  logger.Debug("There were no matching articles")
	return nil, nil
}

func sendPutRequest(article Article, id int) (*http.Response, error) {
  // Put request logic
	base_url, exists := os.LookupEnv("BASE_DOMAIN")
	if !exists {
		return nil, fmt.Errorf("Base url does not exists, please set the BASE_DOMAIN env variable")
	}
	post_endpoint, exists := os.LookupEnv("ENDPOINT")
	if !exists {
		return nil, fmt.Errorf("Endpoint not provided, please set the ENDPOINT env variable")
	}

	env := os.Getenv("ENV")
	if len(env) == 0 {
		env = "PROD"
	}
	protocol := "https://"
	if env == "DEV" {
		protocol = "http://"
	}

  url := protocol + base_url + "/" + post_endpoint + fmt.Sprintf("%d/", id)
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
	//logger.Debug(fmt.Sprintf("Article post request: %v", string(body)))
	// send post request
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	req.Header.Add("Authorization", "Basic "+basicAuth)
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending article update request: %v", err)
	}

	return resp, nil
}
// Send POST request cont. article payload to site
func sendPostRequest(article Article) (*http.Response, error) {
	base_url, exists := os.LookupEnv("BASE_DOMAIN")
	if !exists {
		return nil, fmt.Errorf("Base url does not exists, please set the BASE_DOMAIN env variable")
	}
	post_endpoint, exists := os.LookupEnv("ENDPOINT")
	if !exists {
		return nil, fmt.Errorf("Endpoint not provided, please set the ENDPOINT env variable")
	}

	env := os.Getenv("ENV")
	if len(env) == 0 {
		env = "PROD"
	}
	protocol := "https://"
	if env == "DEV" {
		protocol = "http://"
	}

	url := protocol + base_url + "/" + post_endpoint
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
	//logger.Debug(fmt.Sprintf("Article post request: %v", string(body)))
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

// Creates an article object that can be sent via a POST requests
func createArticlePayload(articleName string, articleFile string, articlePhotos string) (Article, error) {
	data, err := os.ReadFile(articleFile)
	if err != nil {
		return Article{}, fmt.Errorf("Error reading file: %v", err)
	}
	title := strings.ReplaceAll(articleName, "_", " ")

	//content := strings.ReplaceAll(string(data), "\r\n", " ")
	//content = strings.ReplaceAll(content, "\n", " ")
	content := strings.TrimSpace(string(data))

	//Parse images
	imageFiles, err := os.ReadDir(articlePhotos)
	if articlePhotos == "" {

	} else if os.IsNotExist(err) {
		logger.Debug("No images attached to article")
		return Article{}, err
	}
	var images []Image
	var attachedImages []string
	for _, image := range imageFiles {
		logger.Debug(fmt.Sprintf("Create image paycload for image %v", image))
		imagePayload, err := createImagePayload(filepath.Join(articlePhotos, image.Name()))
		// Over here, there is a potential to send nil data images and upload them later.
		if err != nil {
			logger.Debug(fmt.Sprintf("There was an error creating payload for image, skipping. Error: %v", err))
			continue
		}
		//ext := filepath.Ext(image.Name())
		//imageName := strings.TrimSuffix(image.Name(), ext)
		logger.Debug("Image payload created and added to images")
		attachedImages = append(attachedImages, imagePayload.Filename)
		images = append(images, imagePayload)
	}
	logger.Debug(fmt.Sprintf("Images to be sent are: %v", attachedImages))
	logger.Debug(fmt.Sprintf("Successfully created article payload for %v", articleName))
	return Article{Title: title, Content: content, Images: images, Path: filepath.Dir(articleFile)}, nil
}

func createImagePayload(imageFile string) (Image, error) {
	image, err := os.Stat(imageFile)
	if err != nil {
		return Image{}, fmt.Errorf("Error getting image info: %v", err)
	}
	logger.Debug(fmt.Sprintf("Checking image %v", imageFile))
	ext := filepath.Ext(image.Name())
	imageName := strings.TrimSuffix(image.Name(), ext)
	logger.Debug(fmt.Sprintf(`Image File: %v
    Image Extension: %v
    Image Name: %v`, imageFile, ext, imageName))
	raw_data, err := os.ReadFile(imageFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Image{}, fmt.Errorf("Error: Image %v does not exist", imageFile)
		}
		return Image{Filename: imageName, Data: nil}, nil
	}
	logger.Debug("Checking if image is a valid image file")
	if !checkIfImage(raw_data) {
		logger.Info(fmt.Sprintf("Provided image file: %v is not a valid image, image will be added with dummy data", imageFile))
		return Image{Filename: imageName, Data: nil}, nil
	}
	logger.Debug("Image is valid")
	data := b64.StdEncoding.EncodeToString(raw_data)
	return Image{Filename: imageName, Data: &data}, nil
}

func sendImageUpdate(url string, image Image) (*http.Response, error) {
	//base_url, exists := os.LookupEnv("BASE_DOMAIN")
	//if !exists {
	//	return nil, fmt.Errorf("Base url does not exists, please set the BASE_DOMAIN env variable")
	//}
	//post_endpoint, exists := os.LookupEnv("ENDPOINT")
	//if !exists {
	//	return nil, fmt.Errorf("Endpoint not provided, please set the ENDPOINT env variable")
	//}

	//env := os.Getenv("ENV")
	//if len(env) == 0 {
	//	env = "PROD"
	//}
	//protocol := "https://"
	//if env == "DEV" {
	//	protocol = "http://"
	//}

	logger.Debug(fmt.Sprintf("Sending request to: %v", url))
	// Need to handle authentication, this can be just simple authentication in our case.
	user := os.Getenv("USERNAME")
	pass := os.Getenv("PASSWORD")
	auth := user + ":" + pass
	basicAuth := b64.StdEncoding.EncodeToString([]byte(auth))

	body, err := json.Marshal(image)
	if err != nil {
		return nil, fmt.Errorf("There was an error formatting the post body: %v", err)
	}
	//logger.Debug(fmt.Sprintf("Article post request: %v", string(body)))
	// send post request
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	req.Header.Add("Authorization", "Basic "+basicAuth)
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending article creation request: %v", err)
	}

	return resp, nil
}

func uploadArticleImages(imageUrls []string, images []Image) {
	for i, imageUrl := range imageUrls {
		// We essentailly need to send a post request to our returned url so that an image can be uploaded
		logger.Debug(fmt.Sprintf("Uploading image %v with image data", images[i].Filename))
		fmt.Print(imageUrl)
		fmt.Print(images[i])
		resp, err := sendImageUpdate(imageUrl, images[i])
		if err != nil {
			logger.Debug(fmt.Sprintf("There was an error updating image at %v", imageUrl))
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Debug(fmt.Sprintf("There was an error reading the response body from the update action"))
			continue
		}
		logger.Info("Image upload response", "status", resp.Status, "body", string(body))
	}
}
