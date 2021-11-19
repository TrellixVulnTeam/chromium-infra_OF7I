# GoFindit
GoFindit is the culprit finding service for compile and test failures for Chrome and ChromeOS.

This is the rewrite in Golang of the Python2 version of Findit (findit-for-me.appspot.com).

## Local Development
To run the server locally, firstly you need to authenticate
```
gcloud auth application-default login
```
and
```
luci-auth login -scopes "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email"
```

After that, run
```
go run main.go
```

This will start a web server running at http://localhost:8800