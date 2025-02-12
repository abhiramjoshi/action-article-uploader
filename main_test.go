package main

import (
	"log"
	"path/filepath"
	"reflect"
	"testing"
)

func compareImages(i1 Image, i2 Image) bool {
  if i1.Filename != i2.Filename {
    return false
  }

  if i2.Data != i2.Data {
    return false
  }
  return true
}

func compareArticles(a1 Article, a2 Article) bool {
  a1Values := reflect.ValueOf(a1)
  a1Types := a1Values.Type()
  a2Values := reflect.ValueOf(a2)
  log.Printf("Num of fields: %d", a1Values.NumField())
  
  for i := 0;i < a1Values.NumField(); i++ {
    log.Printf("Considering article value: %v", a1Types.Field(i).Name)
    if a1Types.Field(i).Name != "Images" {
      val1 := a1Values.Field(i).Interface()
      val2 := a2Values.Field(i).Interface()
      log.Printf("a1: %v, a2: %v", val1, val2)
      log.Printf("a1 and a2 are equal: %v", val1 == val2)
      if val1 != val2 {
        log.Printf("Values are not equal")
        return false
      }
    } else {
      i1 := a1.Images
      i2 := a2.Images

      if len(i1) != len(i2) {
        return false
      }
      for j := 0; j < len(i1); j++ {
        if !compareImages(i1[j], i2[j]) {
          return false 
        }
      }
      log.Printf("Images are equal")
    }
  }
  return true
}

func TestParseArticle(t *testing.T) {
  articleFolder := "test"
  wantName := "testing"
  wantArticle := "test/testing.md"
  wantPhotos := "test/photos"
  name, article, photos, err := parseArticle(articleFolder)

  if wantName != name || wantArticle != article || wantPhotos != photos || err != nil {
    t.Fatalf(`parseArticle("%v") = %q, %q, %q, %v, but need %q, %q, %q`, 
              articleFolder, 
              name, 
              article, 
              photos, 
              err, 
              wantName, 
              wantArticle, 
              wantPhotos)
  }
}

func TestParseArticleNoPhotos(t *testing.T) {
  articleFolder := "./test1"
  wantName := "testing"
  wantArticle := "test1/testing.md"
  wantPhotos := ""
  name, article, photos, err := parseArticle(articleFolder)

  if wantName != name || wantArticle != article || wantPhotos != photos || err != nil {
    t.Fatalf(`parseArticle("%v") = %q, %q, %q, %v, but need %q, %q, %q`, articleFolder, name, article, photos, err, wantName, wantArticle, wantPhotos)
  }
}

func TestParseArticleNoArticle(t *testing.T) {
  articleFolder := "./test2"
  wantName := ""
  wantArticle := ""
  wantPhotos := ""
  name, article, photos, err := parseArticle(articleFolder)

  if wantName != name || wantArticle != article || wantPhotos != photos || err == nil {
    t.Fatalf(`parseArticle("%v") = %q, %q, %q, %v, but need %q, %q, %q`, articleFolder, name, article, photos, err, wantName, wantArticle, wantPhotos)
  }
}

func TestCreateArticleStruct(t *testing.T) {
  articleFolder := "./test"
  wantContent := "This is a test article ![testing](testimage)"
  wantImages := []Image{{Filename: "testimage", Data: ""}}
  wantArticleStruct := Article{Title: "testing", Content: wantContent, Images: wantImages, Path: filepath.Clean(articleFolder)}
  name, article, photos, err := parseArticle(articleFolder)
  if err != nil {
    t.Fatalf("There was an error: %v", err)
  }
  artStruct, err := createArticlePayload(name, article, photos)
  log.Printf("%v", artStruct)
  if !compareArticles(artStruct, wantArticleStruct) || err != nil {
    t.Fatalf(`createArticlePayload(%v) resulted in an unexpected output.\n Wanted: %v\nGot: %v`, articleFolder, artStruct, wantArticleStruct)
  }
}

func TestPostArticle(t *testing.T) {
  
}
