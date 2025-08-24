# Article Uploader

This go module, designed to be used as a CLI tool will allow us to automatically detect and upload articles found in our articles directory to our website. The usefulness here is that it allows us to offload the work of creating an API request, sending it to our server and handling any errors to a self-contianed bit of code, written in a more advanced and maintainable language that the jobs available on Github Actions purely

### Operation Flow Overall

1. Github Actions detects change in files in article folder
2. Github Actions calls article_uploader giving it the folder of the article that changed
3.
    1. Go program figures out if article exists, modifies if it does
    2. If article does not exist, Go program will create the article by sending an API request to the server to upload the article
4. Response to article has image links, therefore images will be uploaded
5. Go writes log out for visibility

### Operational Flow Go

1. Go gets article folder
2. Uses folder name to determine if an article exists or not
3. Article is flattened and sent in a POST request to the server
4. Server responds with list of pictures that need to be uploaded
5. Go uploads each picture if it exists in the image directory of the article
6. Finished, any cleanup required is complete

### To build
There is an automated CICD implemented in Github actions that will build a new `go` binary when changes are detected in `go` files, and push this binary to the bins directory. These files are than available for use in the actions

To trigger this CICD build. Simply make changes to go files, specify a new version in the `.build_version` file and push the changes. 

To skip a build on commit, push the commit with: `[skip actions]` in the commit message
